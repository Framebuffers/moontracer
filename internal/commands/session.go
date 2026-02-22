package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type session struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *session) Data() *discordgo.ApplicationCommand { return nil }

// Execute is the logic that runs when the user invokes that command.
func (r *session) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
