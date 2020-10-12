package main

import (
	"fmt"
	"log"
	"time"
)

var (
	refreshInterval = 60 * time.Second
)

func RunScraper(bot *Bot, db *Db, api *Osuapi, requests chan int) {
	// start timers
	go startTimers(db, requests)

	for userId := range requests {
		log.Println("scraping user", userId)
		newMaps, err := getNewMaps(db, api, userId)
		if err != nil {
			log.Println("err getting new maps:", err)
		}

		db.IterTrackingChannels(userId, func(channelId string) error {
			bot.NotifyNewEvent(channelId, newMaps)
			return nil
		})

		// wait a minute and put them back into the queue
		go func(id int) {
			time.Sleep(refreshInterval)
			requests <- id
		}(userId)
	}
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
