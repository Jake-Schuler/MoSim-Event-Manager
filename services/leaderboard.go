package services

import (
	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
	"gorm.io/gorm"
)

func GetLeaderboard(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}

	// Calculate RP for each user from their matches
	for i := range users {
		winRP, bonusRP, AutoPoints, TeleopPoints, EndgamePoints := calculateUserStats(db, users[i].MMID)
		users[i].WinRP = winRP
		users[i].BonusRP = bonusRP
		users[i].TotalRP = winRP + bonusRP
		users[i].TotalPoints = AutoPoints + TeleopPoints + EndgamePoints
		users[i].AutoPoints = AutoPoints
		users[i].TeleopPoints = TeleopPoints
		users[i].EndgamePoints = EndgamePoints
	}

	// Sort users by TotalRP (descending)
	for i := 0; i < len(users)-1; i++ {
		for j := i + 1; j < len(users); j++ {
			if users[i].TotalRP < users[j].TotalRP {
				users[i], users[j] = users[j], users[i]
			}
		}
	}

	// Assign ranks based on the sorted order
	for i := range users {
		users[i].Rank = i + 1
	}

	return users, nil
}

func calculateUserStats(db *gorm.DB, userMMID int) (int, int, int, int, int) {
	var matches []models.QualsMatch
	if err := db.Where("red_player_id = ? OR blue_player_id = ?", userMMID, userMMID).Find(&matches).Error; err != nil {
		return 0, 0, 0, 0, 0
	}

	totalWinRP := 0
	totalBonusRP := 0
	AutoPoints := 0
	TeleopPoints := 0
	EndgamePoints := 0

	for _, match := range matches {
		if match.RedPlayerID == userMMID {
			// User is on red alliance
			if match.RedScore > match.BlueScore {
				// Red alliance won
				totalWinRP += match.RedWinRP
			}
			// Always add bonus RP regardless of win/loss
			totalBonusRP += match.RedBonusRP
			AutoPoints += match.RedAutoScore
			TeleopPoints += match.RedTeleopScore
			EndgamePoints += match.RedEndgameScore
		} else if match.BluePlayerID == userMMID {
			// User is on blue alliance
			if match.BlueScore > match.RedScore {
				// Blue alliance won
				totalWinRP += match.BlueWinRP
			}
			// Always add bonus RP regardless of win/loss
			totalBonusRP += match.BlueBonusRP
			AutoPoints += match.BlueAutoScore
			TeleopPoints += match.BlueTeleopScore
			EndgamePoints += match.BlueEndgameScore

		}
	}

	return totalWinRP, totalBonusRP, AutoPoints, TeleopPoints, EndgamePoints
}

func GetUserMatches(db *gorm.DB, userMMID int) ([]models.QualsMatch, error) {
	var matches []models.QualsMatch
	if err := db.Where("red_player_id = ? OR blue_player_id = ?", userMMID, userMMID).Find(&matches).Error; err != nil {
		return nil, err
	}
	return matches, nil
}
