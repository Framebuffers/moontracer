package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type campaignEdit struct {
	db *bun.DB
}

func (r *campaignEdit) Data() *discordgo.ApplicationCommand { return nil }

func (r *campaignEdit) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
