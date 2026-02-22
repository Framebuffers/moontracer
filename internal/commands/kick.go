package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type kick struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *kick) Data() *discordgo.ApplicationCommand { return nil }

// Execute is the logic that runs when the user invokes that command.
func (r *kick) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
