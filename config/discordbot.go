package config

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

func InitDiscordBot() *discordgo.Session {
	// Create a new Discord session using the provided token
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		panic("Error creating Discord session: " + err.Error())
	}

	// Open a websocket connection to Discord
	if err := dg.Open(); err != nil {
		panic("Error opening Discord session: " + err.Error())
	}

	dg.UpdateWatchStatus(0, "Robotics")

	return dg
}
