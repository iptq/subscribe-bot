package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"subscribe-bot/config"
	"subscribe-bot/db"
	"subscribe-bot/discord"
	"subscribe-bot/osuapi"
	"subscribe-bot/scrape"
)

var exit_chan = make(chan int)

func main() {
	configPath := flag.String("config", "config.toml", "Path to the config file (defaults to config.toml)")
	flag.Parse()

	config, err := config.ReadConfig(*configPath)

	api := osuapi.New(&config)

	db, err := db.OpenDb("db", api)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("opened db")

	bot, err := discord.NewBot(&config, db, api)
	if err != nil {
		log.Fatal(err)
	}

	go scrape.RunScraper(bot, db, api)

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for {
			s := <-signal_chan
			switch s {
			case syscall.SIGHUP:
				fallthrough
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				fallthrough
			case syscall.SIGQUIT:
				exit_chan <- 0
			default:
				exit_chan <- 1
			}
		}
	}()
	code := <-exit_chan

	db.Close()
	bot.Close()
	scrape.Ticker.Stop()
	os.Exit(code)
}
