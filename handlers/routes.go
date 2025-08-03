package handlers

import (
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB, dg *discordgo.Session) {
	// Public routes
	r.GET("/", HomeHandler(db))
	r.GET("/register", RegisterHandler(db))
	r.GET("/auth", RegisterHandler(db))
	r.GET("/leaderboard", LeaderboardHandler(db))
	r.GET("/ws", WebSocketHandler(db))
	r.GET("/overlay", OverlayHandler())

	// Admin routes
	authorized := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"user": os.Getenv("ADMIN_PASSWORD"),
	}))

	authorized.GET("/", AdminDashboardHandler(db))
	authorized.GET("/users", AdminUsersHandler(db))
	authorized.POST("/toggle_schedule", ToggleScheduleHandler(db))
	authorized.GET("/generate", GenerateMatchesHandler(db))
	authorized.GET("/match/:id/edit", EditMatchesHandler(db))
	authorized.POST("/match/:id/edit", EditMatchesHandler(db))
	authorized.GET("/set_active_match", SetActiveMatchHandler(db, dg))
	authorized.GET("/set_event_name", SetEventNameHandler(db))
	authorized.GET("/toggle_leaderboard", ToggleLeaderboardVisibilityHandler(db))
	authorized.GET("/allianceSelection", AllianceSelectionHandler(db))
	authorized.POST("/allianceSelection", AllianceSelectionHandler(db))
	authorized.POST("/toggle_alliance_selection", ToggleAllianceSelectionHandler(db))
}
