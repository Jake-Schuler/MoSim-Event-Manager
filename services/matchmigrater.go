package services

import (
	"fmt"

	"github.com/Jake-Schuler/MoSim-Event-Manager/config"
	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
)

func MigrateMatchSchedule() error {
	db := config.InitDB()

	matches := ParseMatchSchedule() // Assume this returns ([]models.Match, error)

	for _, match := range matches {
		team1, ok1 := match["team1"].(map[string]interface{})
		team2, ok2 := match["team2"].(map[string]interface{})
		if !ok1 || !ok2 {
			return fmt.Errorf("team1 or team2 is not a map[string]interface{}")
		}

		// Fix: Use lowercase "mmid" and convert int to string
		redMMIDInt, ok1 := team1["mmid"].(int)
		blueMMIDInt, ok2 := team2["mmid"].(int)
		if !ok1 || !ok2 {
			return fmt.Errorf("mmid not found or not an int in team1 or team2")
		}

		if err := db.Create(&models.QualsMatch{
			RedPlayerID:  redMMIDInt,
			BluePlayerID: blueMMIDInt,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
