package main

import (
	"log"
	"os"

	"github.com/framebuffers/moontracer/internal/discord"
)

func main() {
	if os.Getenv("VERBOSE") == "true" {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		log.Println("verbose mode enabled")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN is required")
	}

	guildID := os.Getenv("DISCORD_GUILD_ID")

	adminRole := os.Getenv("ADMIN_ROLE_NAME")
	if adminRole == "" {
		log.Fatal("ADMIN_ROLE_NAME is required")
	}

	modRole := os.Getenv("MOD_ROLE_NAME")

	bot, err := discord.New(token, guildID, adminRole, modRole)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := bot.Run(); err != nil {
		log.Fatalf("bot error: %v", err)
	}
}
