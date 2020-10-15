package scrape

import (
	"fmt"
	"subscribe-bot/osuapi"
)

func (s *Scraper) scrapeNominatedMaps() {
	events, _ := s.api.GetBeatmapsetEvents(&osuapi.GetBeatmapsetEventsOptions{
		Types: []string{"nominate", "qualify"},
	})

	for _, event := range events {
		fmt.Println(event)
	}
}
