package interactions

import (
	"github.com/bwmarrin/discordgo"
)

/*
respondUpdate replaces the current message instead of sending a new one.

Used by back buttons and select menus to update in place.
*/
func respondUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
