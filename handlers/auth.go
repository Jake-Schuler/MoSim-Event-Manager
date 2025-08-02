package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Jake-Schuler/ORC-MatchMaker/models"
	"github.com/Jake-Schuler/ORC-MatchMaker/services"
)

func RegisterHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		access_token := c.Query("access_token")
		preferred_name := c.Query("preferred_name") // Get preferred name from query parameter

		if access_token == "" {
			c.HTML(400, "authRedirect.tmpl", gin.H{
				"error":       "access_token is required",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}

		req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{
				"error":       "Failed to create request",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}
		req.Header.Set("Authorization", "Bearer "+access_token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{
				"error":       "Failed to contact Discord",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			c.HTML(400, "authRedirect.tmpl", gin.H{
				"error":       "Invalid access token",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}

		var userInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{
				"error":       "Failed to parse Discord response",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}

		idStr, ok := userInfo["id"].(string)
		if !ok {
			c.HTML(500, "authRedirect.tmpl", gin.H{
				"error":       "Discord response missing id",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}

		idInt, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{
				"error":       "Invalid Discord user id",
				"ClientID":    os.Getenv("DISCORD_CLIENT_ID"),
				"RedirectURI": os.Getenv("DISCORD_REDIRECT_URI"),
			})
			return
		}

		if db.First(&models.User{ID: idInt}).RowsAffected > 0 {
			c.JSON(200, gin.H{
				"message": "User already registered",
			})
			return
		}

		username, _ := userInfo["username"].(string)

		db.Create(&models.User{
			ID:               idInt,
			Username:         username,
			PreferedUsername: preferred_name, // Set the preferred username
			MMID:             services.CurrentMMID,
		})
		services.CurrentMMID++

		c.JSON(200, gin.H{
			"message": "Registered successfully",
		})
	}
}
