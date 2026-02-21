package commands

import (
	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

type awooCommand struct{}

func (c *awooCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AwooCommandName,
		Description: messages.AwooCommandDesc,
	}
}

func (c *awooCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "awooooooo",
		},
	})
}

