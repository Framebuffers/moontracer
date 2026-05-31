package main

import (
	"log"
	"os"
	"path/filepath"

	// NOTE: this embeds the IANA timezone database so LoadLocation works without system tzdata
	_ "time/tzdata"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/discord"
	"github.com/framebuffers/moontracer/internal/mediaserver"
)

/*
Flow:
 1. Read required env vars: DISCORD_BOT_TOKEN, ADMIN_ROLE_NAME. Optional: MOD_ROLE_NAME.
 2. Read optional DISCORD_GUILD_ID (dev mode: register commands to single guild).
 3. Each guild gets its own SQLite DB in the "data" directory (bind-mounted via Docker).
 4. Create GuildDBManager (per-guild databases are created on demand).
 5. Create Discord bot with token, guild ID, admin role name, and GuildDBManager.
 6. Start the bot- open gateway, discover guilds, init DBs, register commands, listen for interactions.
 7. Block until SIGINT/SIGTERM (Ctrl+C), then clean up and exit.
*/
func main() {
	if os.Getenv("VERBOSE") == "true" {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		log.Println("verbose mode enabled")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN is required")
	}

	// optional: if set, commands register to that guild only (instant); if empty, commands register globally (up to 1h propagation)
	guildID := os.Getenv("DISCORD_GUILD_ID")

	adminRole := os.Getenv("ADMIN_ROLE_NAME")
	if adminRole == "" {
		log.Fatal("ADMIN_ROLE_NAME is required")
	}

	// optional: if empty, mods are admin-assigned only
	modRole := os.Getenv("MOD_ROLE_NAME")

	// init directories: one for data, one for DBs, and another for media served
	dataDir := os.Getenv("MOONTRACER_DATA_DIR")
	dbDir := filepath.Join(dataDir, "db")
	mediaDir := filepath.Join(dataDir, "media")

	for _, dir := range []struct {
		path string
		mode os.FileMode
	}{
		{dataDir, 0750},
		{dbDir, 0700},    // <- owner-only
		{mediaDir, 0750}, // <- media: served.
	} {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			log.Fatalf("failed to create DB directory %s: %v", dbDir, err)
		}
	}

	guildDBM := db.NewGuildDBManager(dbDir)

	mediaPort := os.Getenv("MEDIA_PORT")
	if mediaPort == "" {
		mediaPort = "8090"
	}
	mediaBaseURL := os.Getenv("MEDIA_BASE_URL")
	if mediaBaseURL == "" {
		mediaBaseURL = "http://localhost:" + mediaPort + "/api/v1/cdn"
	}

	downloadURL := os.Getenv("MEDIA_DOWNLOAD_URL")
	if downloadURL == "" {
		downloadURL = "http://localhost:" + mediaPort
	}
	mediaserver.SetDownloadBase(downloadURL)
	mediaserver.Serve(mediaDir, ":"+mediaPort)
	mediaserver.Probe(":" + mediaPort)

	bot, err := discord.New(token, guildID, adminRole, modRole, mediaDir, mediaBaseURL, guildDBM)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := bot.Run(); err != nil {
		log.Fatalf("bot error: %v", err)
	}
}
