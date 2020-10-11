package main

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	*discordgo.Session
	mentionRe *regexp.Regexp
	db        *Db
	requests  chan int
}

func NewBot(token string, db *Db, requests chan int) (bot *Bot, err error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return
	}

	err = s.Open()
	if err != nil {
		return
	}
	log.Println("connected to discord")

	re, err := regexp.Compile("\\s*<@\\!?" + s.State.User.ID + ">\\s*")
	if err != nil {
		return
	}

	bot = &Bot{s, re, db, requests}
	s.AddHandler(bot.errWrap(bot.newMessageHandler))
	return
}

func (bot *Bot) errWrap(fn interface{}) interface{} {
	val := reflect.ValueOf(fn)
	origType := reflect.TypeOf(fn)
	origTypeIn := make([]reflect.Type, origType.NumIn())
	for i := 0; i < origType.NumIn(); i++ {
		origTypeIn[i] = origType.In(i)
	}
	newType := reflect.FuncOf(origTypeIn, []reflect.Type{}, false)
	newFunc := reflect.MakeFunc(newType, func(args []reflect.Value) (result []reflect.Value) {
		res := val.Call(args)
		if len(res) > 0 && !res[0].IsNil() {
			err := res[0].Interface().(error)
			if err != nil {
				msg := fmt.Sprintf("error: %s", err)
				channel, _ := bot.UserChannelCreate("100443064228646912")
				id, _ := bot.ChannelMessageSend(channel.ID, msg)
				log.Println(id, msg)
			}
		}
		return []reflect.Value{}
	})
	return newFunc.Interface()
}

func (bot *Bot) newMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) (err error) {
	mentionsMe := false
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			mentionsMe = true
			break
		}
	}

	if !mentionsMe {
		return
	}

	msg := bot.mentionRe.ReplaceAllString(m.Content, " ")
	msg = strings.Trim(msg, " ")

	parts := strings.Split(msg, " ")
	switch strings.ToLower(parts[0]) {
	case "track":
		if len(parts) < 2 {
			err = errors.New("fucked up")
			return
		}

		var mapperId int
		mapperId, err = strconv.Atoi(parts[1])
		if err != nil {
			return
		}

		err = bot.db.ChannelTrackMapper(m.ChannelID, mapperId, 3)
		if err != nil {
			return
		}

		go func() {
			time.Sleep(refreshInterval)
			bot.requests <- mapperId
		}()

		bot.MessageReactionAdd(m.ChannelID, m.ID, "\xf0\x9f\x91\x8d")
	}

	return
}

func (bot *Bot) Close() {
	bot.Session.Close()
}
