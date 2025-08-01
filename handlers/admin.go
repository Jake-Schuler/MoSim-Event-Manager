package handlers

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/Jake-Schuler/ORC-MatchMaker/services"
)

func AdminDashboardHandler(db *gorm.DB) gin.HandlerFunc {

	return func(c *gin.Context) {
		var match models.QualsMatch
		if err := db.First(&match).Error; err != nil {
			c.JSON(404, gin.H{"error": "Match not found"})
			return
		}

		// Get all users for the dropdown
		var users []models.User
		if err := db.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch users"})
			return
		}
		c.HTML(200, "admin.tmpl", gin.H{
			"title":            "Admin Dashboard",
			"isSchedulePublic": GetSchedulePublic(),
			"matches":          services.ParseMatchScheduleFromDB(),
			"users":            users,
		})
	}
}

func AdminUsersHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		if err := db.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to retrieve users"})
			return
		}
		c.JSON(200, users)
	}
}

func ToggleScheduleHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		SetSchedulePublic(!GetSchedulePublic())
		c.JSON(200, gin.H{"isSchedulePublic": GetSchedulePublic()})
	}
}

func GenerateMatchesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userCount int64
		if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to count users"})
			return
		}

		userCountStr := strconv.FormatInt(userCount, 10)
		numberofmatches := c.Query("numberofmatches")
		if numberofmatches == "" {
			numberofmatches = "1"
		}

		numberofmatchesInt, err := strconv.Atoi(numberofmatches)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid number of matches"})
			return
		}

		totalMatches := numberofmatchesInt * int(userCount)
		if totalMatches == 0 {
			c.JSON(400, gin.H{"error": "Invalid number of matches"})
			return
		}

		cwd, err := os.Getwd()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get current working directory", "details": err.Error()})
			return
		}

		var matchMakerExePath string
		switch runtime.GOOS {
		case "windows":
			matchMakerExePath = "data/MatchMaker.exe"
		default:
			matchMakerExePath = "./data/MatchMaker"
		}

		cmd := exec.Command(matchMakerExePath, "-t", userCountStr, "-r", numberofmatches, "-a", "2", "-s")
		cmd.Env = os.Environ()
		cmd.Dir = cwd

		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{
				"error":    "Failed to run MatchMaker.exe",
				"details":  err.Error(),
				"output":   string(output),
				"path":     matchMakerExePath,
				"cwd_used": cwd,
			})
			return
		}

		os.WriteFile("match_schedule.txt", output, 0644)

		if err := db.Where("1 = 1").Delete(&models.QualsMatch{}).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to clear existing matches", "details": err.Error()})
			return
		}

		if err := db.Exec("UPDATE sqlite_sequence SET seq = 0 WHERE name = 'quals_matches'").Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to reset match sequence", "details": err.Error()})
			return
		}

		services.MigrateMatchSchedule()
		c.JSON(200, gin.H{"message": "Match schedule generated", "output": string(output)})
	}
}
