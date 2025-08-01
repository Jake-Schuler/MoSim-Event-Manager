package models

type User struct {
	ID               int    `gorm:"primaryKey"`
	Username         string `gorm:"uniqueIndex"`
	PreferedUsername string `gorm:"uniqueIndex"`
	MMID             int    `gorm:"uniqueIndex"` // MatchMaker ID
	TotalRP          int    `gorm:"default:0"`   // Total Ranking Points
	WinRP            int    `gorm:"default:0"`   // Win Ranking Points
	BonusRP          int    `gorm:"default:0"`   // Bonus Ranking Points
	TotalPoints      int    `gorm:"default:0"`   // Total Points from matches
	AutoPoints       int    `gorm:"default:0"`   // Auto Points
	TeleopPoints     int    `gorm:"default:0"`   // Teleop Points
	EndgamePoints    int    `gorm:"default:0"`   // Endgame Points
	Rank             int    `gorm:"-"`
}
