package handlers

import (
	"log"
	"net/http"

	"github.com/Jake-Schuler/MoSim-Event-Manager/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func WebSocketHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log the incoming request
		log.Printf("WebSocket upgrade request from %s", c.ClientIP())
		log.Printf("Headers: %v", c.Request.Header)

		conn, err := services.Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not upgrade connection"})
			return
		}

		// Handle the WebSocket connection using the service
		services.HandleWebSocketConnection(conn, db)
	}
}
