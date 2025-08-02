package handlers

import (
	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/Jake-Schuler/ORC-MatchMaker/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AllianceSelectionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle GET request - render the page
		if c.Request.Method == "GET" {
			c.HTML(200, "allianceselection.tmpl", gin.H{
				"title": "Alliance Selection",
			})
			return
		}

		// Handle POST request - process JSON data
		if c.Request.Method == "POST" {
			// Parse JSON request body
			var request struct {
				Alliance  int    `json:"alliance"`
				Captain   string `json:"captain"`
				Selection string `json:"selection"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(400, gin.H{"error": "Invalid JSON data"})
				return
			}

			// Validate alliance number
			if request.Alliance < 1 || request.Alliance > 8 {
				c.JSON(400, gin.H{"error": "Invalid alliance number"})
				return
			}

			// Upsert alliance selection (update if exists, else create)
			allianceSelection := models.AllianceSelection{
				AllianceNumber:    request.Alliance,
				AllianceCaptain:   request.Captain,
				AllianceSelection: request.Selection,
			}
			db.Where(models.AllianceSelection{AllianceNumber: request.Alliance}).Assign(allianceSelection).FirstOrCreate(&allianceSelection)

			services.BroadcastAllianceSelection(models.AllianceSelection{
				AllianceNumber:    request.Alliance,
				AllianceCaptain:   request.Captain,
				AllianceSelection: request.Selection,
			})

			c.JSON(200, gin.H{
				"message": "Alliance selection created successfully",
			})
		}
	}
}

func ToggleAllianceSelectionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		services.ToggleAllianceSelectionVisibility()
		c.JSON(200, gin.H{
			"message": "Alliance selection toggled",
		})
	}
}
