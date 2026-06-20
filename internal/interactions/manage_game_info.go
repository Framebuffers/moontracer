package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		Triggered from the manage-campaign Settings sub-menu via the "Game Info" button.
		1. Button click (manage_game_info:<campaignID>): manageGameInfoHandler
			a. Loads campaign (DM-auth + mutable check).
			b. Opens a modal pre-filled with current Rules, VTT, BooksAllowed, and Extra.
		2. Modal submit (modal_manage_game_info:<campaignID>): manageGameInfoModal
			a. Re-loads + re-checks DM auth + mutable.
			b. Writes Game.Rules, Game.VTT, Game.BooksAllowed (comma-split), and Extra.
			c. db.Update + async billboard refresh + terminal success reply.
*/

type manageGameInfoHandler struct {
	db *bun.DB
}

func (h *manageGameInfoHandler) CustomIDPrefix() string {
	return messages.ManageGameInfoPrefix
}

func (h *manageGameInfoHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	booksValue := strings.Join(campaign.Game.BooksAllowed, ", ")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.ManageGameInfoModalID, campaignID),
			Title:    messages.ManageGameInfoModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "rules",
						Label:       messages.ManageGameInfoRulesLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.ManageGameInfoRulesPlaceholder,
						Value:       campaign.Game.Rules,
						Required:    false,
						MaxLength:   1000,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "vtt",
						Label:       messages.ManageGameInfoVTTLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.ManageGameInfoVTTPlaceholder,
						Value:       campaign.Game.VTT,
						Required:    false,
						MaxLength:   200,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "books",
						Label:       messages.ManageGameInfoBooksLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.ManageGameInfoBooksPlaceholder,
						Value:       booksValue,
						Required:    false,
						MaxLength:   500,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "extra",
						Label:       messages.ManageGameInfoExtraLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.ManageGameInfoExtraPlaceholder,
						Value:       campaign.Extra,
						Required:    false,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
}

type manageGameInfoModal struct {
	db *bun.DB
}

func (h *manageGameInfoModal) CustomIDPrefix() string {
	return messages.ManageGameInfoModalID
}

func (h *manageGameInfoModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	for _, row := range i.ModalSubmitData().Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range ar.Components {
			ti, ok := comp.(*discordgo.TextInput)
			if !ok {
				continue
			}
			switch ti.CustomID {
			case "rules":
				campaign.Game.Rules = strings.TrimSpace(ti.Value)
			case "vtt":
				campaign.Game.VTT = strings.TrimSpace(ti.Value)
			case "books":
				campaign.Game.BooksAllowed = parseCommaList(ti.Value)
			case "extra":
				campaign.Extra = strings.TrimSpace(ti.Value)
			}
		}
	}

	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_game_info: update failed for campaign %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	go func() {
		if err := helpers.UpdateBillboard(s, h.db, campaign); err != nil {
			log.Printf("manage_game_info: billboard update for %s: %v", campaignID, err)
		}
	}()

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageGameInfoSuccess, campaign.Name))
}

// parseCommaList splits a comma-or-newline-separated string into trimmed, non-empty entries.
func parseCommaList(raw string) []string {
	var out []string
	for _, item := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
