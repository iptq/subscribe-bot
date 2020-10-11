package main

// Database is laid out like this:
// mapper/<mapper_id>/trackers/<channel_id> -> priority
// mapper/<mapper_id>/latestEvent
// channel/<channel_id>/tracks/<mapper_id> -> priority

import (
	"strconv"

	bolt "go.etcd.io/bbolt"
)

var (
	LATEST_EVENT = []byte("latestEvent")
)

type Db struct {
	*bolt.DB
	api *Osuapi
}

func OpenDb(path string, api *Osuapi) (db *Db, err error) {
	inner, err := bolt.Open(path, 0666, nil)
	db = &Db{inner, api}
	return
}

// Loop over channels that are tracking this specific mapper
func (db *Db) IterTrackingChannels(mapperId int, fn func(channelId string) error) (err error) {
	err = db.DB.View(func(tx *bolt.Tx) error {
		mapper := getMapper(tx, mapperId)
		if mapper == nil {
			return nil
		}

		trackers := mapper.Bucket([]byte("trackers"))
		if trackers == nil {
			return nil
		}

		c := trackers.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			channelId := string(k)
			err := fn(channelId)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return
}

// Loop over tracked mappers
func (db *Db) IterTrackedMappers(fn func(userId int) error) (err error) {
	err = db.DB.View(func(tx *bolt.Tx) error {
		mappers := tx.Bucket([]byte("mapper"))
		if mappers == nil {
			return nil
		}

		c := mappers.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			userId, err := strconv.Atoi(string(k))
			if err != nil {
				return err
			}

			err = fn(userId)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return
}

// Get a list of channels that are tracking this mapper
func (db *Db) GetMapperTrackers(userId int) (trackersList []string) {
	trackersList = make([]string, 0)
	db.DB.View(func(tx *bolt.Tx) error {
		mapper, err := getMapperMut(tx, userId)
		if err != nil {
			return err
		}

		trackers := mapper.Bucket([]byte("trackers"))
		if trackers == nil {
			return nil
		}

		c := trackers.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			channelId := string(k)
			trackersList = append(trackersList, channelId)
		}

		return nil
	})
	return
}

// Update the latest event of a mapper to the given one
func (db *Db) UpdateMapperLatestEvent(userId int, eventId int) (err error) {
	err = db.DB.Update(func(tx *bolt.Tx) error {
		mapper, err := getMapperMut(tx, userId)
		if err != nil {
			return err
		}

		err = mapper.Put(LATEST_EVENT, []byte(strconv.Itoa(eventId)))
		if err != nil {
			return err
		}

		return nil
	})
	return
}

// Get the latest event ID of this mapper, if they have one
func (db *Db) MapperLastEvent(userId int) (has bool, id int) {
	has = false
	id = -1
	db.DB.View(func(tx *bolt.Tx) error {
		mapper := getMapper(tx, userId)
		if mapper == nil {
			return nil
		}

		lastEventId := mapper.Get(LATEST_EVENT)
		if lastEventId == nil {
			return nil
		}

		var err error
		id, err = strconv.Atoi(string(lastEventId))
		if err != nil {
			return nil
		}

		has = true
		return nil
	})

	return
}

// Start tracking a new mapper (if they're not already tracked)
func (db *Db) ChannelTrackMapper(channelId string, mapperId int, priority int) (err error) {
	events, err := db.api.GetUserEvents(mapperId, 1, 0)
	if err != nil {
		return
	}

	err = db.Batch(func(tx *bolt.Tx) error {
		{
			mapper, err := getMapperMut(tx, mapperId)
			if err != nil {
				return err
			}

			if len(events) > 0 {
				latestEventId := strconv.Itoa(events[0].ID)
				mapper.Put(LATEST_EVENT, []byte(latestEventId))
			}

			trackers, err := mapper.CreateBucketIfNotExists([]byte("trackers"))
			if err != nil {
				return err
			}

			err = trackers.Put([]byte(channelId), []byte(strconv.Itoa(priority)))
			if err != nil {
				return err
			}
		}
		{
			channels, err := tx.CreateBucketIfNotExists([]byte("channels"))
			if err != nil {
				return err
			}

			channel, err := channels.CreateBucketIfNotExists([]byte(channelId))
			if err != nil {
				return err
			}

			tracks, err := channel.CreateBucketIfNotExists([]byte("tracks"))
			if err != nil {
				return err
			}

			err = tracks.Put([]byte(strconv.Itoa(mapperId)), []byte(strconv.Itoa(priority)))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (db *Db) Close() {
	db.DB.Close()
}

func getMapper(tx *bolt.Tx, userId int) (mapper *bolt.Bucket) {
	mappers := tx.Bucket([]byte("mapper"))
	if mappers == nil {
		return nil
	}

	mapper = mappers.Bucket([]byte(strconv.Itoa(userId)))
	if mapper == nil {
		return nil
	}

	return
}

func getMapperMut(tx *bolt.Tx, userId int) (mapper *bolt.Bucket, err error) {
	mappers, err := tx.CreateBucketIfNotExists([]byte("mapper"))
	if err != nil {
		return
	}

	mapper, err = mappers.CreateBucketIfNotExists([]byte(strconv.Itoa(userId)))
	if err != nil {
		return
	}

	return
}
