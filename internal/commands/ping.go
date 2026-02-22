package commands

import (
	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

// pingCommand tests the connectivity between client and server. Responds with 'pong!' when successful.
type pingCommand struct{}

// Data is the command metadata that Discord shows to users.
func (c *pingCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.PingCommandName,
		Description: messages.PingCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (c *pingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "pong!",
		},
	})
}
