package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type session struct {
	db *bun.DB
}

func (r *session) Data() *discordgo.ApplicationCommand { return nil }

func (r *session) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
