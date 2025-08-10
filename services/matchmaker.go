package services

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/Jake-Schuler/MoSim-Event-Manager/config"
	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
)

var CurrentMMID = 1

func GetMMID(db *gorm.DB) {
	var lastUser models.User
	if err := db.Last(&lastUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			CurrentMMID = 1
		} else {
			fmt.Println("Error fetching last user:", err)
			CurrentMMID = 1
		}
	} else {
		CurrentMMID = lastUser.MMID + 1
	}
}

func ParseMatchSchedule() []map[string]interface{} {
	db := config.InitDB()
	var matches []map[string]interface{}

	if _, err := os.Stat("match_schedule.txt"); err != nil {
		return matches
	}

	data, err := os.ReadFile("match_schedule.txt")
	if err != nil {
		fmt.Println("Error reading match schedule:", err)
		return matches
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// First match: 2nd and 4th numbers (index 1 and 3)
		mmid1, err1 := strconv.Atoi(fields[1])
		mmid2, err2 := strconv.Atoi(fields[3])

		var userA, userB models.User
		foundA := db.Where("mm_id = ?", mmid1).First(&userA).Error == nil
		foundB := db.Where("mm_id = ?", mmid2).First(&userB).Error == nil

		match1 := map[string]interface{}{
			"match": len(matches) + 1,
		}
		if err1 == nil && err2 == nil && foundA && foundB {
			match1["team1"] = map[string]interface{}{
				"mmid":              userA.MMID,
				"username":          userA.Username,
				"prefered_username": userA.PreferedUsername,
			}
			match1["team2"] = map[string]interface{}{
				"mmid":              userB.MMID,
				"username":          userB.Username,
				"prefered_username": userB.PreferedUsername,
			}
		} else {
			match1["error"] = fmt.Sprintf("MMID %d or %d not found", mmid1, mmid2)
		}
		matches = append(matches, match1)

		// Second match: 6th and 8th numbers (index 5 and 7)
		mmid3, err3 := strconv.Atoi(fields[5])
		mmid4, err4 := strconv.Atoi(fields[7])

		var userC, userD models.User
		foundC := db.Where("mm_id = ?", mmid3).First(&userC).Error == nil
		foundD := db.Where("mm_id = ?", mmid4).First(&userD).Error == nil

		match2 := map[string]interface{}{
			"match": len(matches) + 1,
		}
		if err3 == nil && err4 == nil && foundC && foundD {
			match2["team1"] = map[string]interface{}{
				"mmid":              userC.MMID,
				"username":          userC.Username,
				"prefered_username": userC.PreferedUsername,
			}
			match2["team2"] = map[string]interface{}{
				"mmid":              userD.MMID,
				"username":          userD.Username,
				"prefered_username": userD.PreferedUsername,
			}
		} else {
			match2["error"] = fmt.Sprintf("MMID %d or %d not found", mmid3, mmid4)
		}
		matches = append(matches, match2)
	}

	return matches
}

func ParseMatchScheduleFromDB() []map[string]interface{} {
	db := config.InitDB()
	var matches []map[string]interface{}

	var qualsMatches []models.QualsMatch
	if err := db.Find(&qualsMatches).Error; err != nil {
		fmt.Println("Error reading matches from database:", err)
		return matches
	}

	for i, match := range qualsMatches {
		var redUser, blueUser models.User
		foundRed := db.Where("mm_id = ?", match.RedPlayerID).First(&redUser).Error == nil
		foundBlue := db.Where("mm_id = ?", match.BluePlayerID).First(&blueUser).Error == nil

		matchData := map[string]interface{}{
			"match": i + 1,
		}

		if foundRed && foundBlue {
			matchData["team1"] = map[string]interface{}{
				"mmid":              redUser.MMID,
				"username":          redUser.Username,
				"prefered_username": redUser.PreferedUsername,
			}
			matchData["team2"] = map[string]interface{}{
				"mmid":              blueUser.MMID,
				"username":          blueUser.Username,
				"prefered_username": blueUser.PreferedUsername,
			}
		} else {
			matchData["error"] = fmt.Sprintf("MMID %d or %d not found", match.RedPlayerID, match.BluePlayerID)
		}

		matches = append(matches, matchData)
	}

	return matches
}
