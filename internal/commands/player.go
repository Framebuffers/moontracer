package commands

import (
	"github.com/bwmarrin/discordgo"
)

type playerCommand struct{}

func (p *playerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "mycampaigns",
		Description: "Get the campaigns you're a player on.",
	}
}

func (p *playerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {

}
