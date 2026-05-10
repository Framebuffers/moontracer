package interactions

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
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
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    messages.SetCoverInstructions,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{helpers.BackRow(router.ViewManageSettings, campaignID)},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
