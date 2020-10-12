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
		log.Println("scraping", userId)
		newMaps, err := getNewMaps(db, api, userId)
		if err != nil {
			log.Println("err getting new maps:", err)
			exit_chan <- 1
		}

		db.IterTrackingChannels(userId, func(channelId string) error {
			bot.NotifyNewEvent(channelId, newMaps)
			return nil
		})

		// wait a minute and put them back into the queue
		go func() {
			time.Sleep(refreshInterval)
			requests <- userId
		}()
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
		log.Printf("last event id for %d is %d\n", userId, lastEventId)
		offset := 0

	loop:
		for {
			log.Println("loading user events from", offset)
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
