package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/commands"
)

// Bot wraps a discordgo session and manages its lifecycle.
type Bot struct {
	session    *discordgo.Session
	guildID    string
	registered []*discordgo.ApplicationCommand
}

// New creates a Bot with the given token and guild ID.
func New(token, guildID string) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &Bot{session: s, guildID: guildID}, nil
}

// Run opens the gateway, registers guild-scoped commands, blocks until
// SIGINT/SIGTERM, then removes commands and closes the session.
func (b *Bot) Run() error {
	b.session.AddHandler(NewHandler(commands.All()))

	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	appID := b.session.State.User.ID
	log.Printf("logged in as %s (app %s)", b.session.State.User.Username, appID)

	if err := b.registerCommands(appID); err != nil {
		return err
	}

	log.Println("bot is running — press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	b.removeCommands(appID)

	return nil
}

func (b *Bot) registerCommands(appID string) error {
	for _, cmd := range commands.All() {
		created, err := b.session.ApplicationCommandCreate(appID, b.guildID, cmd.Data())
		if err != nil {
			return err
		}
		b.registered = append(b.registered, created)
		log.Printf("registered /%s", created.Name)
	}
	return nil
}

func (b *Bot) removeCommands(appID string) {
	for _, cmd := range b.registered {
		if err := b.session.ApplicationCommandDelete(appID, b.guildID, cmd.ID); err != nil {
			log.Printf("failed to remove /%s: %v", cmd.Name, err)
		} else {
			log.Printf("removed /%s", cmd.Name)
		}
	}
}
