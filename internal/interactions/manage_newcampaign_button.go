package interactions

/*
	New Campaign from button handler for the /me hub and /managecampaigns menu.

	Flow:
		1. Player clicks "New Campaign" button.
		2. Auth check: must be a registered player.
		3. Respond with InteractionResponseModal (same modal as /newcampaign slash command).
		4. Modal submission is handled by the existing modalCampaignCreate handler.
*/

import (
	"moontracer/internal/interactions/helpers"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/messages"
)

type manageNewCampaignButton struct {
	db *bun.DB
}

func (h *manageNewCampaignButton) CustomIDPrefix() string {
	return messages.ManageNewCampaignPrefix
}

func (h *manageNewCampaignButton) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	ok, err := auth.Authorize(h.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("manage_newcampaign: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		helpers.RespondNotRegistered(s, i)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.NewCampaignModalCustomID,
			Title:    messages.NewCampaignModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldNameID,
						Label:       messages.FieldNameLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldNamePlaceholder,
						Required:    true,
						MaxLength:   100,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldSlotsID,
						Label:       messages.FieldSlotsLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldSlotsPlaceholder,
						Required:    false,
						MaxLength:   3,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldDescriptionID,
						Label:       messages.FieldSynopsisLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.FieldDescriptionPlaceholder,
						Required:    true,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
}
