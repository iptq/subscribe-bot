package scrape

import (
	"time"

	"subscribe-bot/config"
	"subscribe-bot/db"
	"subscribe-bot/discord"
	"subscribe-bot/osuapi"
)

var (
	refreshInterval = 30 * time.Second
	lastUpdateTime  time.Time
	Ticker          = time.NewTicker(refreshInterval)
)

type Scraper struct {
	config *config.Config
	bot    *discord.Bot
	db     *db.Db
	api    *osuapi.Osuapi
}

func RunScraper(config *config.Config, bot *discord.Bot, db *db.Db, api *osuapi.Osuapi) {
	lastUpdateTime = time.Now()

	scraper := Scraper{config, bot, db, api}

	go func() {
		for ; true; <-Ticker.C {
			scraper.scrapePendingMaps()
			scraper.scrapeNominatedMaps()
		}
	}()
}
