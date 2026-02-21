package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type addPlayer struct {
	db *bun.DB
}

func (r *addPlayer) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AddPlayerCommandName,
		Description: messages.AddPlayerCommandDesc,
	}
}

func (r *addPlayer) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}

