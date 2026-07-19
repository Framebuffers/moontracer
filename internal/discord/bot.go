package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/guard"
)

/*
Bot wraps a discordgo session and manages its lifecycle.

Note: guildID is optional: if set, commands register to that guild only (dev mode).
adminRole and modRole are stored for future use with auth.SyncServerRoles.
*/
type Bot struct {
	session    *discordgo.Session
	guildID    string
	adminRole  string
	modRole    string
	registered []*discordgo.ApplicationCommand
}

// New creates a Bot with the given token, guild ID, and role names.
func New(token, guildID, adminRole, modRole string) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		session:   s,
		guildID:   guildID,
		adminRole: adminRole,
		modRole:   modRole,
	}, nil
}

// Run opens the gateway, registers commands, blocks until SIGINT/SIGTERM, then cleans up.
func (b *Bot) Run() error {
	b.session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	cmds := commands.All()
	b.session.AddHandler(NewHandler(cmds))

	if err := b.session.Open(); err != nil {
		return err
	}
	defer b.session.Close()

	appID := b.session.State.User.ID
	log.Printf("bot: logged in as %s (app %s)", b.session.State.User.Username, appID)

	if err := b.registerCommands(appID, cmds); err != nil {
		return err
	}

	log.Println("bot is running- press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	if b.guildID != "" {
		b.removeCommands(appID)
	} else {
		log.Println("bot: leaving global commands registered across restart")
	}

	return nil
}

func (b *Bot) registerCommands(appID string, all []commands.Command) error {
	// In dev mode, clear stale global registrations from previous runs.
	if guard.DebugGuildID != "" {
		existing, err := b.session.ApplicationCommands(appID, "")
		if err != nil {
			log.Printf("bot: warning: could not list global commands for cleanup: %v", err)
		} else {
			for _, g := range existing {
				if err := b.session.ApplicationCommandDelete(appID, "", g.ID); err != nil {
					log.Printf("bot: warning: failed to delete stale global /%s: %v", g.Name, err)
				} else {
					log.Printf("bot: deleted stale global /%s", g.Name)
				}
			}
		}
	}

	cmdData := make([]*discordgo.ApplicationCommand, len(all))
	for i, cmd := range all {
		cmdData[i] = cmd.Data()
	}

	registered, err := b.session.ApplicationCommandBulkOverwrite(appID, b.guildID, cmdData)
	if err != nil {
		return err
	}
	b.registered = registered

	scope := "global"
	if b.guildID != "" {
		scope = "guild " + b.guildID
	}
	for _, cmd := range registered {
		log.Printf("bot: registered /%s (%s)", cmd.Name, scope)
	}
	return nil
}

func (b *Bot) removeCommands(appID string) {
	for _, cmd := range b.registered {
		if err := b.session.ApplicationCommandDelete(appID, b.guildID, cmd.ID); err != nil {
			log.Printf("bot: failed to remove /%s: %v", cmd.Name, err)
		} else {
			log.Printf("bot: removed /%s", cmd.Name)
		}
	}
}
