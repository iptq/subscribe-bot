package web

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"subscribe-bot/config"
)

func RunWeb(config *config.Config) {
	r := gin.Default()
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	addr := fmt.Sprintf("%s:%d", config.Web.Host, config.Web.Port)
	r.Run(addr)
}
