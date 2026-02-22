package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type addPlayer struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *addPlayer) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AddPlayerCommandName,
		Description: messages.AddPlayerCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (r *addPlayer) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
