package main

import (
	"log"
	"os"

	"moontracer/internal/db"
	"moontracer/internal/discord"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}

	guildID := os.Getenv("DISCORD_GUILD_ID")
	if guildID == "" {
		log.Fatal("DISCORD_GUILD_ID is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "moontracer.db"
	}

	var dbm db.DatabaseManager
	bunDB, err := dbm.Get(dbPath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	bot, err := discord.New(token, guildID, bunDB)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := bot.Run(); err != nil {
		log.Fatalf("bot error: %v", err)
	}
}
