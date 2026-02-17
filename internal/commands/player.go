package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type playerCommand struct {
	db *bun.DB
}

func (p *playerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "mycampaigns",
		Description: "Get the campaigns you're a player on.",
	}
}

func (p *playerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {

}
