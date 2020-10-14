package web

import (
	"errors"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (web *Web) mapVersions(c *gin.Context) {
	userId := c.Param("userId")
	mapId := c.Param("mapId")

	repoDir := path.Join(web.config.Repos, userId, mapId)
	repo, _ := git.PlainOpen(repoDir)

	type Revision struct {
		Date      time.Time
		HumanDate string
		Summary   string
		Hash      string
		HasParent bool
	}

	versions := make([]Revision, 0)
	logIter, _ := repo.Log(&git.LogOptions{})
	for i := 0; i < 20; i++ {
		commit, err := logIter.Next()
		if err == io.EOF {
			break
		}

		stats, _ := commit.Stats()
		_, err = commit.Parent(0)
		hasParent := !errors.Is(err, object.ErrParentNotFound)

		versions = append(versions, Revision{
			Date:      commit.Author.When,
			HumanDate: humanize.Time(commit.Author.When),
			Summary:   stats.String(),
			Hash:      commit.Hash.String(),
			HasParent: hasParent,
		})
	}

	c.HTML(http.StatusOK, "map-version.html", gin.H{
		"LoggedIn": isLoggedIn(c),
		"Versions": versions,
	})
}

func (web *Web) mapPatch(c *gin.Context) {
	userId := c.Param("userId")
	mapId := c.Param("mapId")
	hash := c.Param("hash")

	repoDir := path.Join(web.config.Repos, userId, mapId)
	repo, _ := git.PlainOpen(repoDir)

	hashObj := plumbing.NewHash(hash)
	commit, _ := repo.CommitObject(hashObj)
	parent, _ := commit.Parent(0)
	patch, _ := commit.Patch(parent)

	c.String(http.StatusOK, "text/plain", patch.String())
}
