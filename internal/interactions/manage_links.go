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
		Triggered from the manage-campaign Settings sub-menu via the "Edit Links" button.
		1. Button click (manage_links:<campaignID>): manageLinksHandler
			a. Loads the campaign (DM-auth + mutable check).
			b. Opens a modal pre-filled with current VTTLink and Links (newline-joined).
			   Two fields: vtt_link (short) and resources (paragraph).
		2. Modal submit (manage_links_modal:<campaignID>): manageLinksModal
			a. Re-loads + re-checks DM auth + mutable.
			b. Writes campaign.VTTLink (trimmed) and campaign.Links (parseLinks splits
			   newlines, trims, drops blanks).
			c. db.Update + terminal success reply.

	Notes:
		- PlayerSheetURL is set elsewhere (campaign_announce.go modal); this file
		  only manages VTT + free-form resource links.
		- parseLinks lives here because it's the only call site.
*/

type manageLinksHandler struct {
	db *bun.DB
}

func (h *manageLinksHandler) CustomIDPrefix() string {
	return messages.ManageLinksPrefix
}

func (h *manageLinksHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	resourcesValue := strings.Join(campaign.Links, "\n")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.ManageLinksModalID, campaignID),
			Title:    messages.ManageLinksModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "vtt_link",
						Label:       messages.ManageLinksVTTLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.ManageLinksVTTPlaceholder,
						Value:       campaign.VTTLink,
						Required:    false,
						MaxLength:   500,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "resources",
						Label:       messages.ManageLinksResourcesLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.ManageLinksResourcesPlaceholder,
						Value:       resourcesValue,
						Required:    false,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
}

type manageLinksModal struct {
	db *bun.DB
}

func (h *manageLinksModal) CustomIDPrefix() string {
	return messages.ManageLinksModalID
}

func (h *manageLinksModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	fields := i.ModalSubmitData().Components
	for _, row := range fields {
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
			case "vtt_link":
				campaign.VTTLink = strings.TrimSpace(ti.Value)
			case "resources":
				campaign.Links = parseLinks(ti.Value)
			}
		}
	}

	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_links: update failed for campaign %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageLinksSuccess, campaign.Name))
}

// parseLinks splits a newline-separated string into a slice of non-empty trimmed URLs.
func parseLinks(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		if u := strings.TrimSpace(line); u != "" {
			out = append(out, u)
		}
	}
	return out
}
