package handlers

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
	"github.com/Jake-Schuler/MoSim-Event-Manager/services"
)

func AdminDashboardHandler(db *gorm.DB) gin.HandlerFunc {

	return func(c *gin.Context) {
		// Get all users for the dropdown
		var users []models.User
		if err := db.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch users"})
			return
		}

		// Try to get first match, but don't fail if none exist
		var match models.QualsMatch
		hasMatches := db.First(&match).Error == nil

		c.HTML(200, "admin.tmpl", gin.H{
			"title":            "Admin Dashboard",
			"isSchedulePublic": GetSchedulePublic(),
			"matches":          services.ParseMatchScheduleFromDB(),
			"users":            users,
			"hasMatches":       hasMatches,
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

func SetActiveMatchHandler(db *gorm.DB, dg *discordgo.Session) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get match level and ID from query parameters
		matchLevel := c.Query("level")
		matchIDStr := c.Query("id")
		if matchIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing match ID"})
			return
		}

		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid match ID"})
			return
		}

		var match models.QualsMatch
		err = db.Where("id = ?", matchID).First(&match).Error
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		switch matchLevel {
		case "Quals":
			// Broadcast the active match update to all WebSocket clients
			services.BroadcastActiveMatch(
				matchLevel,
				matchID,
				strconv.Itoa(match.RedPlayerID),
				strconv.Itoa(match.BluePlayerID),
				db,
			)

			var redPlayer, bluePlayer models.User

			// Find users by MMID
			db.Where("mm_id = ?", match.RedPlayerID).First(&redPlayer)
			db.Where("mm_id = ?", match.BluePlayerID).First(&bluePlayer)

			dg.ChannelMessageSend(
				os.Getenv("DISCORD_CHANNEL_ID"),
				"Quals "+strconv.Itoa(matchID)+" will be <@"+strconv.Itoa(redPlayer.ID)+"> vs. <@"+strconv.Itoa(bluePlayer.ID)+">")
		case "Playoffs":
			services.BroadcastActiveMatch(
				matchLevel,
				matchID,
				"",
				"",
				db,
			)
		}

		c.Redirect(http.StatusSeeOther, "/admin")
		// Return success response
		c.JSON(http.StatusOK, gin.H{
			"message":      "Active match set and broadcasted",
			"level":        matchLevel,
			"matchID":      matchID,
			"RedAlliance":  match.RedPlayerID,
			"BlueAlliance": match.BluePlayerID,
		})
	}
}

func ShowEndgameScreenHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchIDStr := c.Param("id")
		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid match ID"})
			return
		}

		var match models.QualsMatch
		if err := db.First(&match, matchID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		var redUser, blueUser models.User
		if err := db.Where("mm_id = ?", match.RedPlayerID).First(&redUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Red player not found"})
			return
		}
		if err := db.Where("mm_id = ?", match.BluePlayerID).First(&blueUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Blue player not found"})
			return
		}

		services.EndScreenBroadcast(
			[]string{redUser.PreferedUsername},
			[]string{blueUser.PreferedUsername},
		)
		c.Redirect(http.StatusSeeOther, "/admin")
	}
}

func ToggleScheduleHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		SetSchedulePublic(!GetSchedulePublic())
		c.JSON(200, gin.H{"isSchedulePublic": GetSchedulePublic()})
	}
}

func ToggleLeaderboardVisibilityHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		services.ToggleLeaderboardVisibility()
		c.JSON(200, gin.H{"message": "Leaderboard visibility toggled"})
	}
}

func SetEventNameHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventName := c.Query("eventName")
		if eventName == "" {
			c.JSON(400, gin.H{"error": "Event name cannot be empty"})
			return
		}

		services.SetEventName(eventName) // Update the global variable using a setter
		c.JSON(200, gin.H{"message": "Event name updated", "eventName": eventName})
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
