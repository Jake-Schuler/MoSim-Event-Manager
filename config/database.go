package config

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/Jake-Schuler/MoSim-Event-Manager/models"
)

func InitDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("data/event.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.QualsMatch{})
	db.AutoMigrate(&models.AllianceSelection{})
	return db
}
