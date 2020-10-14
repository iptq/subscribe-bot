package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

func (web *Web) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("access_token")
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (web *Web) login(c *gin.Context) {
	url := url.URL{
		Scheme: "https",
		Host:   "osu.ppy.sh",
		Path:   "/oauth/authorize",
	}
	q := url.Query()
	q.Set("client_id", web.config.Oauth.ClientId)
	q.Set("redirect_uri", web.config.Web.ServedAt+"/login/callback")
	q.Set("response_type", "code")
	q.Set("scope", "identify public")
	q.Set("state", "urmom")
	url.RawQuery = q.Encode()
	fmt.Println("redirecting to", url.String())
	c.Redirect(http.StatusTemporaryRedirect, url.String())
}

func (web *Web) loginCallback(c *gin.Context) {
	receivedCode := c.Query("code")

	bodyQuery := url.Values{}
	bodyQuery.Set("client_id", web.config.Oauth.ClientId)
	bodyQuery.Set("client_secret", web.config.Oauth.ClientSecret)
	bodyQuery.Set("code", receivedCode)
	bodyQuery.Set("grant_type", "authorization_code")
	bodyQuery.Set("redirect_uri", web.config.Web.ServedAt+"/login/callback")
	body := strings.NewReader(bodyQuery.Encode())
	resp, _ := web.hc.Post("https://osu.ppy.sh/oauth/token", "application/x-www-form-urlencoded", body)
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
}
