package web

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/kofalt/go-memoize"

	"subscribe-bot/config"
	"subscribe-bot/osuapi"
)

const (
	USER_KEY = "user"
)

var (
	cache = memoize.NewMemoizer(90*time.Second, 10*time.Minute)
)

type Web struct {
	config  *config.Config
	api     *osuapi.Osuapi
	hc      *http.Client
	version string
}

func RunWeb(config *config.Config, api *osuapi.Osuapi, version string) {
	hc := &http.Client{
		Timeout: 10 * time.Second,
	}

	web := Web{config, api, hc, version}
	web.Run()
}

func (web *Web) Run() {
	if !web.config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(static.Serve("/static", static.LocalFile("web/static", false)))
	r.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte(web.config.Web.SessionSecret))))

	r.HTMLRender = ginview.New(goview.Config{
		Root:         "web/templates",
		Master:       "master.html",
		DisableCache: web.config.Debug,
		Funcs: template.FuncMap{
			"GitCommit": func() string {
				return web.version
			},
		},
	})

	r.GET("/logout", web.logout)
	r.GET("/login", web.login)
	r.GET("/login/callback", web.loginCallback)

	r.GET("/map/:userId/:mapId/versions", web.mapVersions)
	r.GET("/map/:userId/:mapId/patch/:hash", web.mapPatch)
	r.GET("/map/:userId/:mapId/zip/:hash", web.mapZip)

	r.GET("/", func(c *gin.Context) {
		beatmapSets := web.listRepos()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"LoggedIn":    isLoggedIn(c),
			"Beatmapsets": beatmapSets,
		})
	})

	addr := fmt.Sprintf("%s:%d", web.config.Web.Host, web.config.Web.Port)
	r.Run(addr)
}

func isLoggedIn(c *gin.Context) bool {
	session := sessions.Default(c)
	var accessToken string
	loggedIn := false
	accessTokenI := session.Get("access_token")
	if accessTokenI != nil {
		accessToken = accessTokenI.(string)
		if len(accessToken) > 0 {
			loggedIn = true
		}
	}
	return loggedIn
}

func (web *Web) listRepos() []osuapi.Beatmapset {
	expensive := func() (interface{}, error) {
		repos := make([]int, 0)
		reposDir := web.config.Repos
		users, _ := ioutil.ReadDir(reposDir)

		for _, user := range users {
			userDir := path.Join(reposDir, user.Name())
			var maps []os.FileInfo
			maps, _ = ioutil.ReadDir(userDir)

			for _, mapId := range maps {
				mapDir := path.Join(userDir, mapId.Name())
				fmt.Println(mapDir)

				id, _ := strconv.Atoi(mapId.Name())
				repos = append(repos, id)
			}
		}

		beatmapSets := make([]osuapi.Beatmapset, len(repos))
		var wg sync.WaitGroup
		for i, repo := range repos {
			wg.Add(1)
			go func(i int, repo int) {
				bs, _ := web.api.GetBeatmapSet(repo)
				beatmapSets[i] = bs
				wg.Done()
			}(i, repo)
		}
		wg.Wait()

		return beatmapSets, nil
	}

	result, _, _ := cache.Memoize("key1", expensive)
	return result.([]osuapi.Beatmapset)
}
