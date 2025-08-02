package handlers

import (
	"github.com/gin-gonic/gin"
)

func OverlayHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, "obsoverlay.tmpl", gin.H{
			"title": "Overlay",
		})
	}
}
