package handlers

import (
	"strconv"

	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/Jake-Schuler/ORC-MatchMaker/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func EditMatchesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the ID from the URL parameter
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid match ID"})
			return
		}

		var match models.QualsMatch
		if err := db.First(&match, id).Error; err != nil {
			c.JSON(404, gin.H{"error": "Match not found"})
			return
		}

		// Get all users for the dropdown
		var users []models.User
		if err := db.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch users"})
			return
		}

		if c.Request.Method == "GET" {
			c.HTML(200, "editMatch.tmpl", gin.H{
				"title": "Edit Match " + strconv.Itoa(match.ID),
				"match": match,
				"users": users,
			})
			return
		}

		if c.Request.Method == "POST" {
			redAlliance := c.PostForm("redAlliance")
			blueAlliance := c.PostForm("blueAlliance")
			redTeleopScore := c.PostForm("redTeleopScore")
			blueTeleopScore := c.PostForm("blueTeleopScore")
			redAutoScore := c.PostForm("redAutoScore")
			blueAutoScore := c.PostForm("blueAutoScore")
			redEndgameScore := c.PostForm("redEndgameScore")
			blueEndgameScore := c.PostForm("blueEndgameScore")
			redBonusRP := c.PostForm("redBonusRP")
			blueBonusRP := c.PostForm("blueBonusRP")

			redID, err := strconv.Atoi(redAlliance)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red alliance ID"})
				return
			}

			blueID, err := strconv.Atoi(blueAlliance)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue alliance ID"})
				return
			}

			redTeleopScoreInt, err := strconv.Atoi(redTeleopScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red teleop score"})
				return
			}
			blueTeleopScoreInt, err := strconv.Atoi(blueTeleopScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue teleop score"})
				return
			}
			redAutoScoreInt, err := strconv.Atoi(redAutoScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red auto score"})
				return
			}
			blueAutoScoreInt, err := strconv.Atoi(blueAutoScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue auto score"})
				return
			}
			redEndgameScoreInt, err := strconv.Atoi(redEndgameScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red endgame score"})
				return
			}
			blueEndgameScoreInt, err := strconv.Atoi(blueEndgameScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue endgame score"})
				return
			}

			redScoreInt := redTeleopScoreInt + redAutoScoreInt + redEndgameScoreInt
			blueScoreInt := blueTeleopScoreInt + blueAutoScoreInt + blueEndgameScoreInt

			redBonusRPInt, err := strconv.Atoi(redBonusRP)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red bonus RP"})
				return
			}

			blueBonusRPInt, err := strconv.Atoi(blueBonusRP)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue bonus RP"})
				return
			}

			redWinRP, blueWinRP := 0, 0

			if redScoreInt > blueScoreInt {
				redWinRP = 3
				blueWinRP = 0
			} else if redScoreInt < blueScoreInt {
				redWinRP = 0
				blueWinRP = 3
			} else {
				redWinRP = 1
				blueWinRP = 1
			}
			// Update the match
			if err := db.Model(&match).Updates(models.QualsMatch{
				RedPlayerID:      redID,
				BluePlayerID:     blueID,
				RedTeleopScore:   redTeleopScoreInt,
				BlueTeleopScore:  blueTeleopScoreInt,
				RedAutoScore:     redAutoScoreInt,
				BlueAutoScore:    blueAutoScoreInt,
				RedEndgameScore:  redEndgameScoreInt,
				BlueEndgameScore: blueEndgameScoreInt,
				RedScore:         redScoreInt,
				BlueScore:        blueScoreInt,
				RedWinRP:         redWinRP,
				BlueWinRP:        blueWinRP,
				RedBonusRP:       redBonusRPInt,
				BlueBonusRP:      blueBonusRPInt,
			}).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to update match"})
				return
			}

			// Fetch usernames for broadcast
			var redUser, blueUser models.User
			if err := db.Where("mm_id = ?", redID).First(&redUser).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to fetch red user"})
				return
			}
			if err := db.Where("mm_id = ?", blueID).First(&blueUser).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to fetch blue user"})
				return
			}

			services.BroadcastLeaderboardUpdate(db)
			services.EndScreenBroadcast(
				[]string{redUser.PreferedUsername},
				[]string{blueUser.PreferedUsername},
			)

			c.Redirect(302, "/admin/")
		}
	}
}
