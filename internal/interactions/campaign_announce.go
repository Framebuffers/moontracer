package interactions

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
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
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID)
	if err != nil || !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
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
	parts, ok := splitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID)
	if err != nil || !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return
	}

	var message string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			if input.CustomID == messages.AnnounceFieldID {
				message = input.Value
			}
		}
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.ManageCampaignNotFound)
		return
	}

	if campaign.AnnouncementsThreadID != "" {
		content := fmt.Sprintf("**Announcement from <@%s>:**\n\n%s", userID, message)
		if _, err := s.ChannelMessageSend(campaign.AnnouncementsThreadID, content); err != nil {
			log.Printf("campaign_announce: failed to post to thread %s: %v", campaign.AnnouncementsThreadID, err)
			respondInteraction(s, i, messages.AnnounceError)
			return
		}
		respondInteraction(s, i, fmt.Sprintf(messages.AnnouncePostedToThread, campaign.Name))
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("campaign_announce: failed to load players: %v", err)
		respondInteraction(s, i, messages.AnnounceError)
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
			Content: fmt.Sprintf("**[%s]** Announcement from <@%s>:\n\n%s", campaign.Name, userID, message),
		})
		sent++
	}

	if sent == 0 {
		respondInteraction(s, i, messages.AnnounceNoMembers)
		return
	}

	respondInteraction(s, i, fmt.Sprintf(messages.AnnounceSentMessage, sent, campaign.Name))
}
