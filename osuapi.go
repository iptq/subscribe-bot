package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

const BASE_URL = "https://osu.ppy.sh/api/v2"

type Osuapi struct {
	httpClient   *http.Client
	lock         *semaphore.Weighted
	token        string
	expires      time.Time
	clientId     int
	clientSecret string
}

func NewOsuapi(config *Config) *Osuapi {
	client := &http.Client{
		Timeout: 9 * time.Second,
	}

	// want to cap at around 1000 requests a minute, OSU cap is 1200
	lock := semaphore.NewWeighted(1000)
	return &Osuapi{client, lock, "", time.Now(), config.ClientId, config.ClientSecret}
}

func (api *Osuapi) Token() (token string, err error) {
	if time.Now().Before(api.expires) {
		token = api.token
		return
	}

	data := fmt.Sprintf(
		"client_id=%d&client_secret=%s&grant_type=client_credentials&scope=public",
		api.clientId,
		api.clientSecret,
	)

	resp, err := http.Post(
		"https://osu.ppy.sh/oauth/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data),
	)
	if err != nil {
		return
	}

	var osuToken OsuToken
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(respBody, &osuToken)
	if err != nil {
		return
	}

	log.Println("got new access token", osuToken.AccessToken[:12]+"...")
	api.token = osuToken.AccessToken
	api.expires = time.Now().Add(time.Duration(osuToken.ExpiresIn) * time.Second)
	token = api.token
	return
}

func (api *Osuapi) Request0(action string, url string) (resp *http.Response, err error) {
	err = api.lock.Acquire(context.TODO(), 1)
	if err != nil {
		return
	}
	apiUrl := BASE_URL + url
	req, err := http.NewRequest(action, apiUrl, nil)

	token, err := api.Token()
	if err != nil {
		return
	}

	req.Header.Add("Authorization", "Bearer "+token)
	if err != nil {
		return
	}

	resp, err = api.httpClient.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		var respBody []byte
		respBody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		err = fmt.Errorf("not 200: %s", string(respBody))
		return
	}

	// release the lock after 1 minute
	go func() {
		time.Sleep(time.Minute)
		api.lock.Release(1)
	}()
	return
}

func (api *Osuapi) Request(action string, url string, result interface{}) (err error) {
	resp, err := api.Request0(action, url)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, result)
	if err != nil {
		return
	}

	return
}

func (api *Osuapi) DownloadSingleBeatmap(beatmapId int, path string) (err error) {
	url := fmt.Sprintf("https://osu.ppy.sh/osu/%d", beatmapId)
	resp, err := api.httpClient.Get(url)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return
	}
	return
}

func (api *Osuapi) GetBeatmapSet(beatmapSetId int) (beatmapSet Beatmapset, err error) {
	url := fmt.Sprintf("/beatmapsets/%d", beatmapSetId)
	err = api.Request("GET", url, &beatmapSet)
	if err != nil {
		return
	}

	return
}

func (api *Osuapi) BeatmapsetDownload(beatmapSetId int) (path string, err error) {
	url := fmt.Sprintf("/beatmapsets/%d/download", beatmapSetId)
	resp, err := api.Request0("GET", url)
	if err != nil {
		return
	}

	file, err := ioutil.TempFile(os.TempDir(), "beatmapsetDownload")
	if err != nil {
		return
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return
	}
	file.Close()

	path = file.Name()
	return
}

func (api *Osuapi) GetUser(userId string) (user User, err error) {
	url := fmt.Sprintf("/users/%s", userId)
	err = api.Request("GET", url, &user)
	if err != nil {
		return
	}

	return
}

func (api *Osuapi) GetUserEvents(userId int, limit int, offset int) (events []Event, err error) {
	url := fmt.Sprintf(
		"/users/%d/recent_activity?limit=%d&offset=%d",
		userId,
		limit,
		offset,
	)
	err = api.Request("GET", url, &events)
	if err != nil {
		return
	}

	return
}

func (api *Osuapi) SearchBeatmaps(rankStatus string) (beatmapSearch BeatmapSearch, err error) {
	values := url.Values{}
	values.Set("s", rankStatus)
	query := values.Encode()
	url := "/beatmapsets/search?" + query
	err = api.Request("GET", url, &beatmapSearch)
	if err != nil {
		return
	}

	return
}

type OsuToken struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}
