package commands

import "github.com/bwmarrin/discordgo"

type pingCommand struct{}

func (c *pingCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Replies with pong!",
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
