package commands

import (
	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

type pingCommand struct{}

func (c *pingCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.PingCommandName,
		Description: messages.PingCommandDesc,
	}
}

func (c *pingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong!",
		},
	})
}
