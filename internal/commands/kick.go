package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type kick struct {
	db *bun.DB
}

func (r *kick) Data() *discordgo.ApplicationCommand { return nil }

func (r *kick) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
