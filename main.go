package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var exit_chan = make(chan int)

func main() {
	configPath := flag.String("config", "config.toml", "Path to the config file (defaults to config.toml)")
	flag.Parse()

	config, err := ReadConfig(*configPath)

	requests := make(chan int)

	api := NewOsuapi(&config)

	db, err := OpenDb("db", api)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := NewBot(&config, db, requests)
	if err != nil {
		log.Fatal(err)
	}

	go RunScraper(bot, db, api, requests)

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
	os.Exit(code)
}
