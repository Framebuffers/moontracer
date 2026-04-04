package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/commands"
	"moontracer/internal/dispatch"
	"moontracer/internal/interactions"
)

// Bot wraps a discordgo session and manages its lifecycle.
type Bot struct {
	session    *discordgo.Session
	guildID    string
	db         *bun.DB
	role       string
	registered []*discordgo.ApplicationCommand
	dispatcher *dispatch.Dispatcher
}

// New creates a Bot with the given token, guild ID, admin role name, and database connection.
func New(token, guildID, adminRole string, db *bun.DB) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &Bot{session: s, guildID: guildID, db: db, role: adminRole}, nil
}

/*
	Flow (Run):
		1. Add the main event handler to the session (command dispatcher + component/modal router).
		2. Open the Discord gateway connection.
		3. Log in and get the app ID.
		4. Register all commands globally with Discord (creates /, /campaign, /newcampaign, etc.).
		5. Block waiting for SIGINT/SIGTERM (Ctrl+C).
		6. On shutdown signal, remove all registered commands from Discord.
		7. Close the gateway connection and return.
*/

// Run opens the gateway, registers global commands, blocks until
// SIGINT/SIGTERM, then removes commands and closes the session.
func (b *Bot) Run() error {
	b.dispatcher = dispatch.NewDispatcher(b.session, 5)

	b.session.AddHandler(NewHandler(
		commands.All(b.db, b.dispatcher),
		interactions.AllComponents(b.db, b.guildID, b.role, b.dispatcher),
		interactions.AllModals(b.db, b.guildID, b.role, b.dispatcher),
	))

	b.session.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
		if err := auth.SyncServerRoles(b.db, s, e.GuildID, b.role); err != nil {
			log.Printf("bot: warning: role sync on member update failed: %v", err)
		}
	})

	b.session.AddHandler(HandleGuildMemberRemove(b.db))

	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	b.dispatcher.Start()

	appID := b.session.State.User.ID
	log.Printf("bot: logged in as %s (app %s)", b.session.State.User.Username, appID)

	if err := b.registerCommands(appID); err != nil {
		return err
	}

	if b.guildID != "" {
		if err := auth.SyncServerRoles(b.db, b.session, b.guildID, b.role); err != nil {
			log.Printf("bot: warning: failed to sync server roles: %v", err)
		} else {
			log.Println("bot: server roles synced from Discord")
		}
	}

	log.Println("bot is running — press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	b.dispatcher.Stop()
	b.removeCommands(appID)

	return nil
}

func (b *Bot) registerCommands(appID string) error {
	for _, cmd := range commands.All(b.db, b.dispatcher) {
		created, err := b.session.ApplicationCommandCreate(appID, "", cmd.Data())
		if err != nil {
			return err
		}
		b.registered = append(b.registered, created)
		log.Printf("bot: registered /%s (global)", created.Name)
	}
	return nil
}

func (b *Bot) removeCommands(appID string) {
	for _, cmd := range b.registered {
		if err := b.session.ApplicationCommandDelete(appID, "", cmd.ID); err != nil {
			log.Printf("bot: failed to remove /%s: %v", cmd.Name, err)
		} else {
			log.Printf("bot: removed /%s", cmd.Name)
		}
	}
}
