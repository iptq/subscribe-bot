package scrape

import (
	"fmt"
	"log"
	"time"

	"subscribe-bot/config"
	"subscribe-bot/db"
	"subscribe-bot/discord"
	"subscribe-bot/osuapi"
)

var (
	refreshInterval = 30 * time.Second
	Ticker          = time.NewTicker(refreshInterval)
)

func RunScraper(config *config.Config, bot *discord.Bot, db *db.Db, api *osuapi.Osuapi) {
	lastUpdateTime := time.Now()
	go func() {
		for ; true; <-Ticker.C {
			// build a list of currently tracked mappers
			trackedMappers := make(map[int]int)
			db.IterTrackedMappers(func(userId int) error {
				trackedMappers[userId] = 1
				return nil
			})

			// TODO: is this sorted for sure??
			pendingSets, err := api.SearchBeatmaps("pending")
			if err != nil {
				log.Println("error fetching pending sets", err)
			}

			allNewMaps := make(map[int][]osuapi.Beatmapset, 0)
			var newLastUpdateTime = time.Unix(0, 0)
			for _, beatmapSet := range pendingSets.Beatmapsets {
				updatedTime, err := time.Parse(time.RFC3339, beatmapSet.LastUpdated)
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

				mapperId := beatmapSet.UserId
				if _, ok := trackedMappers[mapperId]; ok {
					if _, ok2 := allNewMaps[mapperId]; !ok2 {
						allNewMaps[mapperId] = make([]osuapi.Beatmapset, 0)
					}

					allNewMaps[mapperId] = append(allNewMaps[mapperId], beatmapSet)
				}
			}

			if len(allNewMaps) > 0 {
				for mapperId, newMaps := range allNewMaps {
					channels := make([]string, 0)
					db.IterTrackingChannels(mapperId, func(channelId string) error {
						channels = append(channels, channelId)
						return nil
					})

					err := bot.NotifyNewBeatmap(channels, newMaps)
					if err != nil {
						log.Println("error notifying new maps:", err)
					}
				}
			}

			lastUpdateTime = newLastUpdateTime
			// this rings the terminal bell when it's updated so i don't have to stare
			// at a blank screen for 30 seconds waiting for the feed to update
			if config.Debug {
				fmt.Print("\a")
			}
			log.Println("last updated time", lastUpdateTime)
		}
	}()
}

func getNewMaps(db *db.Db, api *osuapi.Osuapi, userId int) (newMaps []osuapi.Event, err error) {
	// see if there's a last event
	hasLastEvent, lastEventId := db.MapperLastEvent(userId)
	newMaps = make([]osuapi.Event, 0)
	var (
		events            []osuapi.Event
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
