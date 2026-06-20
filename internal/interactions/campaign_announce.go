package interactions

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/cooldown"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. DM opens `/managecampaigns`, selects a campaign, clicks "Announce".
		2. `manageCampaignAnnounce` catches `manage_announce:<campaignID>`, authorizes (ScopeDM).
		3. Opens a modal with a single text field for the announcement message.
		4. DM submits the modal, triggering `manage_announce_modal:<campaignID>`.
		5. `manageCampaignAnnounceModal` parses the message, loads all active CampaignPlayers.
		6. For each active member (excluding the DM), pushes a DirectMessage via the dispatcher.
		7. Responds to the DM ephemerally: "Announcement sent to X members of Y."
*/

/*
manageCampaignAnnounce opens a modal for the DM to type an announcement.

Custom ID format: manage_announce:<campaignID>
*/
type manageCampaignAnnounce struct {
	db *bun.DB
}

func (h *manageCampaignAnnounce) CustomIDPrefix() string {
	return messages.AnnounceComponentPrefix
}

func (h *manageCampaignAnnounce) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.AnnounceModalPrefix + ":" + campaignID,
			Title:    messages.AnnounceModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.AnnounceFieldID,
						Label:       messages.AnnounceFieldLabel,
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Placeholder: messages.AnnounceFieldPlaceholder,
					},
				}},
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
						Value:       strings.Join(campaign.Links, "\n"),
						Required:    false,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
}

/*
manageCampaignAnnounceModal handles the modal submission and DMs all active members.

	Custom ID format: manage_announce_modal:<campaignID>
*/
type manageCampaignAnnounceModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *manageCampaignAnnounceModal) CustomIDPrefix() string {
	return messages.AnnounceModalPrefix
}

/*
HandleModal processes and handles information to be presented inside the modal screen.

Note:

	The modal follows a Thread-first path. If there's one, post announcements on that thread.
*/
func (h *manageCampaignAnnounceModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}

	cdKey := "announce:" + campaignID
	if !cooldown.Global.Allow(cdKey, 15*time.Minute) {
		remaining := cooldown.Global.Remaining(cdKey)
		helpers.Respond(s, i, fmt.Sprintf(messages.AnnounceCooldown, cooldown.FormatRemaining(remaining)))
		return
	}

	var message string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.AnnounceFieldID:
				message = input.Value
			case "vtt_link":
				campaign.VTTLink = strings.TrimSpace(input.Value)
			case "resources":
				campaign.Links = parseLinks(input.Value)
			}
		}
	}

	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("campaign_announce: failed to save links for campaign %s: %v", campaignID, err)
		// Non-fatal: continue with the announcement even if link save fails.
	}

	if campaign.AnnouncementsThreadID != "" {
		// ensure the thread is locked
		if err := guard.LockThread(s, campaign.AnnouncementsThreadID); err != nil {
			log.Printf("campaign_announce: lock announcements thread %s: %v", campaign.AnnouncementsThreadID, err)
		}

		rolePing := ""
		if campaign.RoleID != "" {
			rolePing = fmt.Sprintf("<@&%s> ", campaign.RoleID)
		}
		content := fmt.Sprintf(messages.AnnounceThreadContent, rolePing, userID, message) + buildAnnounceLinkBlock(campaign, "")
		if _, err := guard.ChannelMessageSend(s, campaign.AnnouncementsThreadID, content); err != nil {
			log.Printf("campaign_announce: failed to post to thread %s: %v", campaign.AnnouncementsThreadID, err)
			helpers.Respond(s, i, messages.AnnounceError)
			return
		}
		helpers.Respond(s, i, fmt.Sprintf(messages.AnnouncePostedToThread, campaign.Name))
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("campaign_announce: failed to load players: %v", err)
		helpers.Respond(s, i, messages.AnnounceError)
		return
	}

	/*
		When DEBUG_ADMIN_ID matches the sender, include them in the recipient list
		so a solo tester can verify announcement DMs.
	*/
	skipSelf := guard.DebugAdminID == "" || guard.DebugAdminID != userID

	msgID := uuid.NewString()
	sent := 0
	for _, p := range players {
		if p.Status != models.StatusActive {
			continue
		}
		if p.PlayerID == userID && skipSelf {
			continue
		}
		h.dispatcher.Push(dispatch.DirectMessage{
			ID:      msgID,
			Sender:  userID,
			Target:  p.PlayerID,
			Content: fmt.Sprintf(messages.AnnounceDMContent, campaign.Name, userID, message) + buildAnnounceLinkBlock(campaign, p.SheetURL),
		})
		sent++
	}

	if sent == 0 {
		helpers.Respond(s, i, messages.AnnounceNoMembers)
		return
	}

	helpers.Respond(s, i, fmt.Sprintf(messages.AnnounceSentMessage, sent, campaign.Name))
}

func buildAnnounceLinkBlock(campaign *models.Campaign, sheetURL string) string {
	if campaign.VTTLink == "" && sheetURL == "" && len(campaign.Links) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(messages.ManageReminderLinks)
	if campaign.VTTLink != "" {
		b.WriteString(fmt.Sprintf(messages.ManageReminderVTT, campaign.VTTLink))
	}
	if sheetURL != "" {
		b.WriteString(fmt.Sprintf(messages.ManageReminderSheets, sheetURL))
	}
	for _, r := range campaign.Links {
		b.WriteString(fmt.Sprintf(messages.ManageReminderResource, r))
	}
	return b.String()
}
