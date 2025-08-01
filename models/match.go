package models

type QualsMatch struct {
	ID               int `gorm:"primaryKey"`
	RedPlayerID      int `gorm:"not null"`
	BluePlayerID     int `gorm:"not null"`
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

type PlayoffMatch struct {
	ID           int `gorm:"primaryKey"`
	RedAlliance  int `gorm:"not null"`
	BlueAlliance int `gorm:"not null"`
	RedScore     int
	BlueScore    int
}
