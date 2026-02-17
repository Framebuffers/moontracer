package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type myCampaigns struct {
	db *bun.DB
}

func (r *myCampaigns) Data() *discordgo.ApplicationCommand { return nil }

func (r *myCampaigns) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {}
