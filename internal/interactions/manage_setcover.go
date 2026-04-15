package interactions

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

/*
manageSetCover is a cosmetic entry point for setting a campaign cover.

Clicking it just surfaces ephemeral instructions pointing at /campaignupload.
The actual upload pipeline lives on the slash command — no plumbing here.
*/
type manageSetCover struct {
	db *bun.DB
}

func (h *manageSetCover) CustomIDPrefix() string {
	return "manage_setcover"
}

func (h *manageSetCover) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messages.SetCoverInstructions,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
