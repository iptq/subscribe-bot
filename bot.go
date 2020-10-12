package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Bot struct {
	*discordgo.Session
	mentionRe *regexp.Regexp
	db        *Db
	api       *Osuapi
	requests  chan int
	config    *Config
}

func NewBot(config *Config, db *Db, requests chan int) (bot *Bot, err error) {
	s, err := discordgo.New("Bot " + config.BotToken)
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

	bot = &Bot{s, re, db, db.api, requests, config}
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

func (bot *Bot) NotifyNewBeatmap(channels []string, newMaps []Beatmapset) (err error) {
	for _, beatmapSet := range newMaps {
		var eventTime time.Time
		eventTime, err = time.Parse(time.RFC3339, beatmapSet.LastUpdated)
		if err != nil {
			return
		}

		var (
			gotDownloadedBeatmap = false
			downloadedBeatmap    BeatmapsetDownloaded
			// status               git.Status

			commit     *object.Commit
			parent     *object.Commit
			patch      *object.Patch
			foundPatch = false
			// commitFiles *object.FileIter
		)
		// beatmapSet, err = bot.getBeatmapsetInfo(beatmap)

		// try to open a repo for this beatmap
		var repo *git.Repository
		repoDir := path.Join(bot.config.Repos, strconv.Itoa(beatmapSet.ID))
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			os.MkdirAll(repoDir, 0777)
		}
		repo, err = git.PlainOpen(repoDir)
		if err == git.ErrRepositoryNotExists {
			// create a new repo
			repo, err = git.PlainInit(repoDir, false)
		}
		if err != nil {
			return
		}

		// download latest updates to the map
		err = bot.downloadBeatmapTo(&beatmapSet, repo, repoDir)
		if err != nil {
			log.Println("failed to download beatmap:", err)
		} else {
			gotDownloadedBeatmap = true
		}

		// create a commit
		var (
			worktree *git.Worktree
			files    []os.FileInfo
			hash     plumbing.Hash
		)
		worktree, err = repo.Worktree()
		if err != nil {
			return
		}
		files, err = ioutil.ReadDir(repoDir)
		if err != nil {
			return
		}
		for _, f := range files {
			if f.Name() == ".git" {
				continue
			}
			worktree.Add(f.Name())
		}
		hash, err = worktree.Commit(
			fmt.Sprintf("update: %d", beatmapSet.ID),
			&git.CommitOptions{
				Author: &object.Signature{
					Name:  beatmapSet.Creator,
					Email: "nobody@localhost",
					When:  eventTime,
				},
			},
		)
		if err != nil {
			err = fmt.Errorf("couldn't create commit for %s: %w", beatmapSet.ID, err)
			return
		}

		commit, err = repo.CommitObject(hash)
		if err != nil {
			err = fmt.Errorf("couldn't find commit with hash %s: %w", hash, err)
			return
		}
		parent, err = commit.Parent(0)
		if errors.Is(err, object.ErrParentNotFound) {
			err = nil
		} else if err != nil {
			err = fmt.Errorf("couldn't retrieve commit parent: %w", err)
			return
		} else {
			patch, err = commit.Patch(parent)
			if err != nil {
				err = fmt.Errorf("couldn't retrieve patch: %w", err)
				return
			}
			foundPatch = true
		}

		embed := &discordgo.MessageEmbed{
			URL:       fmt.Sprintf("https://osu.ppy.sh/s/%d", beatmapSet.ID),
			Title:     fmt.Sprintf("Update: %s - %s", beatmapSet.Artist, beatmapSet.Title),
			Timestamp: eventTime.Format(time.RFC3339),
			Author: &discordgo.MessageEmbedAuthor{
				URL:  "https://osu.ppy.sh/u/" + strconv.Itoa(beatmapSet.UserId),
				Name: beatmapSet.Creator,
				IconURL: fmt.Sprintf(
					"https://a.ppy.sh/%d?%d.png",
					beatmapSet.UserId,
					time.Now().Unix(),
				),
			},
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: beatmapSet.Covers.SlimCover2x,
			},
		}

		if gotDownloadedBeatmap {
			log.Println(downloadedBeatmap)
			if foundPatch {
				embed.Description = fmt.Sprintf(
					"Latest revision: %s\n%s",
					hash,
					patch.Stats().String(),
				)
			} else {
				embed.Description = "Newly tracked map; diff information will be reported upon next update!"
			}
		}

		for _, channelId := range channels {
			_, err = bot.ChannelMessageSendEmbed(channelId, embed)
			if err != nil {
				err = fmt.Errorf("failed to send to %s: %w", channelId, err)
				return
			}
		}
	}

	return
}

type BeatmapsetDownloaded struct {
	Path string
}

func (bot *Bot) downloadBeatmapTo(beatmapSet *Beatmapset, repo *git.Repository, repoDir string) (err error) {
	// clear all OSU files
	files, err := ioutil.ReadDir(repoDir)
	if err != nil {
		return
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".osu") {
			continue
		}
		os.Remove(f.Name())
	}

	for _, beatmap := range beatmapSet.Beatmaps {
		path := path.Join(repoDir, fmt.Sprintf("%d.osu", beatmap.ID))

		err = bot.api.DownloadSingleBeatmap(beatmap.ID, path)
		if err != nil {
			return
		}
	}
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

		var mapper User
		mapperName := strings.Join(parts[1:], " ")
		mapper, err = bot.api.GetUser(mapperName)
		if err != nil {
			return
		}
		mapperId := mapper.ID

		err = bot.db.ChannelTrackMapper(m.ChannelID, mapperId, 3)
		if err != nil {
			return
		}

		// go func() {
		// 	time.Sleep(refreshInterval)
		// 	bot.requests <- mapperId
		// }()

		bot.ChannelMessageSend(m.ChannelID, fmt.Sprintf("subscribed to %+v", mapper))
	}

	return
}

func (bot *Bot) Close() {
	bot.Session.Close()
}
