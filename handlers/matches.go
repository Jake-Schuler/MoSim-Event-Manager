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
			redTotalScore := c.PostForm("redTotalScore")
			blueTotalScore := c.PostForm("blueTotalScore")
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

			redTotalScoreInt, err := strconv.Atoi(redTotalScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid red total score"})
				return
			}
			blueTotalScoreInt, err := strconv.Atoi(blueTotalScore)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid blue total score"})
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

			// Calculate teleop scores
			redTeleopScoreInt := redTotalScoreInt - redAutoScoreInt - redEndgameScoreInt
			blueTeleopScoreInt := blueTotalScoreInt - blueAutoScoreInt - blueEndgameScoreInt

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

			if redTotalScoreInt > blueTotalScoreInt {
				redWinRP = 3
				blueWinRP = 0
			} else if redTotalScoreInt < blueTotalScoreInt {
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
				RedScore:         redTotalScoreInt,
				BlueScore:        blueTotalScoreInt,
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

			c.Redirect(302, "/admin/")
		}
	}
}

// MatchWithNames represents a match with player names instead of IDs
type MatchWithNames struct {
	ID               int
	RedPlayerName    string
	BluePlayerName   string
	RedTeleopScore   int
	BlueTeleopScore  int
	RedAutoScore     int
	BlueAutoScore    int
	RedEndgameScore  int
	BlueEndgameScore int
	RedScore         int
	BlueScore        int
	RedWinRP         int
	BlueWinRP        int
	RedBonusRP       int
	BlueBonusRP      int
}

func MatchResultsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var matches []models.QualsMatch
		if err := db.Find(&matches).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch matches"})
			return
		}

		// Convert matches to include player names
		var matchesWithNames []MatchWithNames
		for _, match := range matches {
			var redUser, blueUser models.User

			// Get red player name
			if err := db.Where("mm_id = ?", match.RedPlayerID).First(&redUser).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to fetch red player"})
				return
			}

			// Get blue player name
			if err := db.Where("mm_id = ?", match.BluePlayerID).First(&blueUser).Error; err != nil {
				c.JSON(500, gin.H{"error": "Failed to fetch blue player"})
				return
			}

			// Use preferred username if available, otherwise use username
			redPlayerName := redUser.Username
			if redUser.PreferedUsername != "" {
				redPlayerName = redUser.PreferedUsername
			}

			bluePlayerName := blueUser.Username
			if blueUser.PreferedUsername != "" {
				bluePlayerName = blueUser.PreferedUsername
			}

			matchWithNames := MatchWithNames{
				ID:               match.ID,
				RedPlayerName:    redPlayerName,
				BluePlayerName:   bluePlayerName,
				RedTeleopScore:   match.RedTeleopScore,
				BlueTeleopScore:  match.BlueTeleopScore,
				RedAutoScore:     match.RedAutoScore,
				BlueAutoScore:    match.BlueAutoScore,
				RedEndgameScore:  match.RedEndgameScore,
				BlueEndgameScore: match.BlueEndgameScore,
				RedScore:         match.RedScore,
				BlueScore:        match.BlueScore,
				RedWinRP:         match.RedWinRP,
				BlueWinRP:        match.BlueWinRP,
				RedBonusRP:       match.RedBonusRP,
				BlueBonusRP:      match.BlueBonusRP,
			}
			matchesWithNames = append(matchesWithNames, matchWithNames)
		}

		if GetSchedulePublic() {
			c.HTML(200, "matchresults.tmpl", gin.H{
				"title":   "Match Results",
				"matches": matchesWithNames,
			})
		} else {
			c.JSON(401, gin.H{"error": "Match results are not public"})
		}
	}
}
