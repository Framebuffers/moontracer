package main

import (
	"log"
	"os"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/discord"
)

/*
	Flow:
		1. Read required env vars: DISCORD_TOKEN, DISCORD_GUILD_ID, ADMIN_ROLE_NAME.
		2. Read optional DB_PATH (defaults to "moontracer.db").
		3. Initialize database connection and run migrations (create tables, add missing columns).
		4. Register all slash commands into the database (populate commands table).
		5. Create Discord bot with token, guild ID, admin role name, and DB connection.
		6. Start the bot — open gateway, register global commands with Discord, listen for interactions.
		7. Block until SIGINT/SIGTERM (Ctrl+C), then clean up and exit.
*/

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}

	guildID := os.Getenv("DISCORD_GUILD_ID") // optional: if set, commands register to that guild only (instant); if empty, commands register globally (up to 1h propagation)

	adminRole := os.Getenv("ADMIN_ROLE_NAME")
	if adminRole == "" {
		log.Fatal("ADMIN_ROLE_NAME is required")
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

	if err := db.Migrate(bunDB); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := commands.RegisterCommands(bunDB); err != nil {
		log.Fatalf("failed to register commands: %v", err)
	}

	bot, err := discord.New(token, guildID, adminRole, bunDB)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := bot.Run(); err != nil {
		log.Fatalf("bot error: %v", err)
	}
}
