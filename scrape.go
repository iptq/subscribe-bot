package main

import (
	"fmt"
	"log"
	"time"
)

var (
	refreshInterval = 30 * time.Second
	ticker          = time.NewTicker(refreshInterval)
)

func RunScraper(bot *Bot, db *Db, api *Osuapi, requests chan int) {
	lastUpdateTime := time.Now()
	go func() {
		for range ticker.C {
			// build a list of currently tracked mappers
			trackedMappers := make(map[int]int)
			db.IterTrackedMappers(func(userId int) error {
				trackedMappers[userId] = 1
				return nil
			})

			// TODO: is this sorted for sure??
			pendingSets, err := bot.api.SearchBeatmaps("pending")
			if err != nil {
				log.Println("error fetching pending sets", err)
			}

			allNewMaps := make(map[int][]Beatmapset, 0)
			var newLastUpdateTime = time.Unix(0, 0)
			for _, beatmap := range pendingSets.Beatmapsets {
				updatedTime, err := time.Parse(time.RFC3339, beatmap.LastUpdated)
				if err != nil {
					log.Println("error parsing last updated time", updatedTime)
				}

				if updatedTime.After(newLastUpdateTime) {
					// update lastUpdateTime to latest updated map
					newLastUpdateTime = updatedTime
				}

				if !updatedTime.After(lastUpdateTime) {
					break
				}

				if mapperId, ok := trackedMappers[beatmap.UserId]; ok {
					if _, ok2 := allNewMaps[mapperId]; !ok2 {
						allNewMaps[mapperId] = make([]Beatmapset, 0)
					}

					allNewMaps[mapperId] = append(allNewMaps[mapperId], beatmap)
				}
			}

			if len(allNewMaps) > 0 {
				log.Println("all new maps", allNewMaps)
				for mapperId, newMaps := range allNewMaps {
					channels := make([]string, 0)
					db.IterTrackingChannels(mapperId, func(channelId string) error {
						channels = append(channels, channelId)
						return nil
					})
					log.Println(newMaps)

					err := bot.NotifyNewBeatmap(channels, newMaps)
					if err != nil {
						log.Println("error notifying new maps", err)
					}
				}
			}

			lastUpdateTime = newLastUpdateTime
			log.Println("last updated time", lastUpdateTime)
		}
	}()
}

func getNewMaps(db *Db, api *Osuapi, userId int) (newMaps []Event, err error) {
	// see if there's a last event
	hasLastEvent, lastEventId := db.MapperLastEvent(userId)
	newMaps = make([]Event, 0)
	var (
		events            []Event
		newLatestEvent    = 0
		updateLatestEvent = false
	)
	if hasLastEvent {
		offset := 0

	loop:
		for {
			events, err = api.GetUserEvents(userId, 50, offset)
			if err != nil {
				err = fmt.Errorf("couldn't load events for user %d, offset %d: %w", userId, offset, err)
				return
			}
			if len(events) == 0 {
				break
			}

			for _, event := range events {
				if event.ID == lastEventId {
					break loop
				}

				if event.ID > newLatestEvent {
					updateLatestEvent = true
					newLatestEvent = event.ID
				}

				if event.Type == "beatmapsetUpload" ||
					event.Type == "beatmapsetRevive" ||
					event.Type == "beatmapsetUpdate" {
					newMaps = append(newMaps, event)
				}
			}

			offset += len(events)
		}
	} else {
		log.Printf("no last event id found for %d\n", userId)
		events, err = api.GetUserEvents(userId, 50, 0)
		if err != nil {
			return
		}

		for _, event := range events {
			if event.ID > newLatestEvent {
				updateLatestEvent = true
				newLatestEvent = event.ID
			}

			if event.Type == "beatmapsetUpload" ||
				event.Type == "beatmapsetRevive" ||
				event.Type == "beatmapsetUpdate" {
				newMaps = append(newMaps, event)
			}
		}
	}

	// TODO: debug
	// updateLatestEvent = false

	if updateLatestEvent {
		err = db.UpdateMapperLatestEvent(userId, newLatestEvent)
		if err != nil {
			return
		}
	}

	return
}

func startTimers(db *Db, requests chan int) {
	db.IterTrackedMappers(func(userId int) error {
		requests <- userId
		return nil
	})
}
