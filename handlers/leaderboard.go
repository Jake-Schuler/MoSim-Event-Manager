package handlers

import (
	"github.com/Jake-Schuler/ORC-MatchMaker/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func LeaderboardHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		leaderboard, err := services.GetLeaderboard(db)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch leaderboard"})
			return
		}
		c.HTML(200, "leaderboard.tmpl", gin.H{
			"title":       "Leaderboard",
			"users":      leaderboard,
		})
	}
}
