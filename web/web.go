package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"

	"subscribe-bot/config"
)

const (
	USER_KEY = "user"
)

func RunWeb(config *config.Config) {
	hc := http.Client{
		Timeout: 10 * time.Second,
	}

	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(static.Serve("/static", static.LocalFile("web/static", false)))
	r.Use(sessions.Sessions("mysession", sessions.NewCookieStore([]byte(config.Web.SessionSecret))))

	r.HTMLRender = ginview.New(goview.Config{
		Root:         "web/templates",
		DisableCache: config.Debug,
	})

	r.GET("/login", func(c *gin.Context) {
		url := url.URL{
			Scheme: "https",
			Host:   "osu.ppy.sh",
			Path:   "/oauth/authorize",
		}
		q := url.Query()
		q.Set("client_id", config.Oauth.ClientId)
		q.Set("redirect_uri", config.Web.ServedAt+"/login/callback")
		q.Set("response_type", "code")
		q.Set("scope", "identify public")
		q.Set("state", "urmom")
		url.RawQuery = q.Encode()
		fmt.Println("redirecting to", url.String())
		c.Redirect(http.StatusTemporaryRedirect, url.String())
	})

	r.GET("/login/callback", func(c *gin.Context) {
		receivedCode := c.Query("code")

		bodyQuery := url.Values{}
		bodyQuery.Set("client_id", config.Oauth.ClientId)
		bodyQuery.Set("client_secret", config.Oauth.ClientSecret)
		bodyQuery.Set("code", receivedCode)
		bodyQuery.Set("grant_type", "authorization_code")
		bodyQuery.Set("redirect_uri", config.Web.ServedAt+"/login/callback")
		body := strings.NewReader(bodyQuery.Encode())
		resp, _ := hc.Post("https://osu.ppy.sh/oauth/token", "application/x-www-form-urlencoded", body)
		respBody, _ := ioutil.ReadAll(resp.Body)
		type OsuToken struct {
			TokenType    string `json:"token_type"`
			ExpiresIn    int    `json:"expires_in"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		var token OsuToken
		_ = json.Unmarshal(respBody, &token)
		fmt.Println("TOKEN", token)

		session := sessions.Default(c)
		session.Set("access_token", token.AccessToken)
		session.Save()

		c.Redirect(http.StatusTemporaryRedirect, "/")
	})

	r.GET("/", func(c *gin.Context) {
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

		repos, _ := getRepos(config)

		// render with master
		c.HTML(http.StatusOK, "index.html", gin.H{
			"LoggedIn": loggedIn,
			"Repos":    repos,
		})
	})

	addr := fmt.Sprintf("%s:%d", config.Web.Host, config.Web.Port)
	r.Run(addr)
}

func getRepos(config *config.Config) (repos []string, err error) {
	repos = make([]string, 0)
	reposDir := config.Repos
	users, err := ioutil.ReadDir(reposDir)
	if err != nil {
		return
	}

	for _, user := range users {
		userDir := path.Join(reposDir, user.Name())
		var maps []os.FileInfo
		maps, err = ioutil.ReadDir(userDir)
		if err != nil {
			return
		}

		for _, mapId := range maps {
			mapDir := path.Join(userDir, mapId.Name())
			fmt.Println(mapDir)
			repos = append(repos, user.Name()+"/"+mapId.Name())
		}
	}

	return
}
