package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type addPlayer struct {
	db *bun.DB
}

func (r *addPlayer) Data() *discordgo.ApplicationCommand { return nil }

func (r *addPlayer) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
