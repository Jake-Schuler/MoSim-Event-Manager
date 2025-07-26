package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type User struct {
	ID       int    `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex"`
	Email    string `gorm:"uniqueIndex"`
	Avatar   string `gorm:"size:255"`
	MMID     int    `gorm:"uniqueIndex"` // MatchMaker ID
}

var CurrentMMID = 1 // Global variable to track the current MMID

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	// Initialize the Gin router
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.StaticFS("/static", http.Dir("static"))
	authorized := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"user": os.Getenv("ADMIN_PASSWORD"),
	}))

	isSchedulePublic := false

	// Connect to the SQLite database
	db, err := gorm.Open(sqlite.Open("users.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&User{})

	r.GET("/", func(c *gin.Context) {
		if !isSchedulePublic {
			c.HTML(200, "index.tmpl", gin.H{
				"title":            "ORC Match Maker",
				"isSchedulePublic": isSchedulePublic,
			})
		} else {
			c.HTML(200, "index.tmpl", gin.H{
				"title":            "ORC Match Maker",
				"matches":          ParseMatchSchedule(),
				"isSchedulePublic": isSchedulePublic,
			})
		}
	})

	r.GET("/register", func(c *gin.Context) {
		access_token := c.Query("access_token")
		if access_token == "" {
			c.HTML(400, "authRedirect.tmpl", gin.H{"error": "access_token is required"})
			return
		}

		req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{"error": "Failed to create request"})
			return
		}
		req.Header.Set("Authorization", "Bearer "+access_token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{"error": "Failed to contact Discord"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			c.HTML(400, "authRedirect.tmpl", gin.H{"error": "Invalid access token"})
			return
		}

		var userInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{"error": "Failed to parse Discord response"})
			return
		}

		idStr, ok := userInfo["id"].(string)
		if !ok {
			c.HTML(500, "authRedirect.tmpl", gin.H{"error": "Discord response missing id"})
			return
		}
		idInt, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(500, "authRedirect.tmpl", gin.H{"error": "Invalid Discord user id"})
			return
		}
		if db.First(&User{ID: idInt}).RowsAffected > 0 {
			c.JSON(200, gin.H{
				"message": "User already registered",
			})
			return
		}

		username, _ := userInfo["username"].(string)
		email, _ := userInfo["email"].(string)
		avatar, _ := userInfo["avatar"].(string)

		db.Create(&User{
			ID:       idInt,
			Username: username,
			Email:    email,
			Avatar:   avatar,
			MMID:     CurrentMMID,
		})
		CurrentMMID++

		c.JSON(200, gin.H{
			"message": "Registered successfully",
		})
	})

	// Admin routes
	authorized.GET("/users", func(c *gin.Context) {
		var users []User
		if err := db.Find(&users).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to retrieve users"})
			return
		}
		c.JSON(200, users)
	})
	authorized.GET("/", func(c *gin.Context) {
		c.HTML(200, "admin.tmpl", gin.H{
			"title":            "Admin Dashboard",
			"isSchedulePublic": isSchedulePublic,
			"matches":          ParseMatchSchedule(),
		})
	})
	authorized.POST("/toggle_schedule", func(c *gin.Context) {
		isSchedulePublic = !isSchedulePublic
		c.JSON(200, gin.H{"isSchedulePublic": isSchedulePublic})
	})
	authorized.GET("/generate", func(c *gin.Context) {
		var userCount int64
		if err := db.Model(&User{}).Count(&userCount).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to count users"})
			return
		}
		userCountStr := strconv.FormatInt(userCount, 10)
		numberofmatches := c.Query("numberofmatches")
		if numberofmatches == "" {
			numberofmatches = "1"
		}

		cwd, err := os.Getwd()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get current working directory", "details": err.Error()})
			return
		}
		var matchMakerExePath string
		switch runtime.GOOS {
		case "windows":
			matchMakerExePath = filepath.Join(cwd, "MatchMaker.exe")
		case "linux":
			matchMakerExePath = filepath.Join(cwd, "MatchMaker")
		}
		fmt.Printf("Attempting to run: %s\n", matchMakerExePath)

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
		ParseMatchSchedule()
		c.JSON(200, gin.H{"message": "Match schedule generated", "output": string(output)})
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		panic("failed to start server")
	}
}

func GetMMID(db *gorm.DB) {
	var lastUser User
	if err := db.Last(&lastUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			CurrentMMID = 1 // Start from 1 if no users exist
		} else {
			fmt.Println("Error fetching last user:", err)
			CurrentMMID = 1 // Fallback to 1 on error
		}
	} else {
		CurrentMMID = lastUser.MMID + 1
	}
}

func ParseMatchSchedule() []map[string]interface{} {
	db, err := gorm.Open(sqlite.Open("users.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}
	var matches []map[string]interface{}
	if _, err := os.Stat("match_schedule.txt"); err == nil {
		data, err := os.ReadFile("match_schedule.txt")
		if err != nil {
			fmt.Println("Error reading match schedule:", err)
			return matches
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				continue // skip incomplete lines
			}
			// First match: 2nd and 4th numbers (index 1 and 3)
			mmid1, err1 := strconv.Atoi(fields[1])
			mmid2, err2 := strconv.Atoi(fields[3])
			var userA, userB User
			foundA := db.First(&userA, "mm_id = ?", mmid1).Error == nil
			foundB := db.First(&userB, "mm_id = ?", mmid2).Error == nil
			match1 := map[string]interface{}{
				"match": len(matches) + 1,
			}
			if err1 == nil && err2 == nil && foundA && foundB {
				match1["team1"] = map[string]interface{}{"mmid": userA.MMID, "username": userA.Username}
				match1["team2"] = map[string]interface{}{"mmid": userB.MMID, "username": userB.Username}
			} else {
				match1["error"] = fmt.Sprintf("MMID %d or %d not found", mmid1, mmid2)
			}
			matches = append(matches, match1)

			// Second match: 6th and 8th numbers (index 5 and 7)
			mmid3, err3 := strconv.Atoi(fields[5])
			mmid4, err4 := strconv.Atoi(fields[7])
			var userC, userD User
			foundC := db.First(&userC, "mm_id = ?", mmid3).Error == nil
			foundD := db.First(&userD, "mm_id = ?", mmid4).Error == nil
			match2 := map[string]interface{}{
				"match": len(matches) + 1,
			}
			if err3 == nil && err4 == nil && foundC && foundD {
				match2["team1"] = map[string]interface{}{"mmid": userC.MMID, "username": userC.Username}
				match2["team2"] = map[string]interface{}{"mmid": userD.MMID, "username": userD.Username}
			} else {
				match2["error"] = fmt.Sprintf("MMID %d or %d not found", mmid3, mmid4)
			}
			matches = append(matches, match2)
		}
	}
	return matches
}
