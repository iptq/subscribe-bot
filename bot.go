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
	api       *Osuapi
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

	bot = &Bot{s, re, db, db.api, requests}
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

func (bot *Bot) NotifyNewEvent(channelId string, newMaps []Event) (err error) {
	for _, event := range newMaps {
		var (
			gotBeatmapInfo       = false
			beatmapSet           Beatmapset
			gotDownloadedBeatmap = false
			downloadedBeatmap    BeatmapsetDownloaded
		)
		beatmapSet, err = bot.getBeatmapsetInfo(event)
		if err != nil {
			log.Println("failed to retrieve beatmap info:", err)
		} else {
			gotBeatmapInfo = true
			downloadedBeatmap, err = bot.downloadBeatmap(&beatmapSet)
			if err != nil {
				log.Println("failed to download beatmap:", err)
			} else {
				gotDownloadedBeatmap = true
			}
		}

		log.Println("BEATMAP SET", beatmapSet)
		embed := &discordgo.MessageEmbed{
			URL:       "https://osu.ppy.sh" + event.Beatmapset.URL,
			Title:     event.Type + ": " + event.Beatmapset.Title,
			Timestamp: event.CreatedAt,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Event ID: %d", event.ID),
			},
		}
		if gotBeatmapInfo {
			embed.Author = &discordgo.MessageEmbedAuthor{
				URL:  "https://osu.ppy.sh/u/" + strconv.Itoa(beatmapSet.UserId),
				Name: beatmapSet.Creator,
				IconURL: fmt.Sprintf(
					"https://a.ppy.sh/%d?%d.png",
					beatmapSet.UserId,
					time.Now().Unix,
				),
			}
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: beatmapSet.Covers.SlimCover2x,
			}

			if gotDownloadedBeatmap {
				log.Println(downloadedBeatmap)
			}
		}
		bot.ChannelMessageSendEmbed(channelId, embed)
	}

	return
}

type BeatmapsetDownloaded struct {
	Path string
}

func (bot *Bot) downloadBeatmap(beatmapSet *Beatmapset) (downloadedBeatmap BeatmapsetDownloaded, err error) {
	beatmapFile, err := bot.api.BeatmapsetDownload(beatmapSet.Id)
	if err != nil {
		return
	}

	downloadedBeatmap.Path = beatmapFile
	return
}

func (bot *Bot) getBeatmapsetInfo(event Event) (beatmapSet Beatmapset, err error) {
	beatmapSetId, err := strconv.Atoi(strings.TrimPrefix(event.Beatmapset.URL, "/s/"))
	if err != nil {
		return
	}

	log.Println("beatmap set id", beatmapSetId)
	beatmapSet, err = bot.api.GetBeatmapSet(beatmapSetId)
	if err != nil {
		return
	}

	return
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
