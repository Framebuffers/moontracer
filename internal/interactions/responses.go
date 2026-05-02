package interactions

import (
	"github.com/bwmarrin/discordgo"
)

/*
respondInteraction sends an ephemeral text response to an interaction.
Used by every handler in this package. It is kept here for discoverability.
*/
func respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
