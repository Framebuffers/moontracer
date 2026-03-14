package commands

import (
	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

/*
	Test command:
		- Checks for connectivity with the VPS/Docker container.
		- Note: the bot uses WebHooks only.
*/

type awooCommand struct{}

// Data is the command metadata that Discord shows to users.
func (c *awooCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AwooCommandName,
		Description: messages.AwooCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (c *awooCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "awooooooo",
		},
	})
}
