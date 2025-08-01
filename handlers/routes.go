package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Public routes
	r.GET("/", HomeHandler(db))
	r.GET("/register", RegisterHandler(db))
	r.GET("/leaderboard", LeaderboardHandler(db))

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
}
