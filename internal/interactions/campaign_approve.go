package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// campaignApprove handles the "Approve" button click on a campaign approval DM.
// Custom ID format: campaign_approve:<campaignID>
type campaignApprove struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (c *campaignApprove) CustomIDPrefix() string {
	return messages.CampaignApprovePrefix
}

func (c *campaignApprove) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, userID, ok := parseApprovalInteraction(s, i)
	if !ok {
		return
	}

	if !checkModAuth(c.db, s, i, userID) {
		return
	}

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignApproveNotFound)
		return
	}

	campaign.IsApproved = true
	if err := db.Update(c.db, campaign); err != nil {
		log.Printf("campaign_approve: failed to approve campaign %s: %v", campaignID, err)
		respondInteraction(s, i, messages.CampaignApproveError)
		return
	}

	msgApproved := &dispatch.DirectMessage{
		ID:      campaignID,
		Sender:  i.User.ID,
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.CampaignApprovedDMMessage, campaign.Name),
	}

	c.dispatcher.Push(*msgApproved)

	respondInteraction(s, i, fmt.Sprintf(messages.CampaignApprovedMessage, campaign.Name))
}

// campaignDeny handles the "Deny" button click on a campaign approval DM.
// Custom ID format: campaign_deny:<campaignID>
type campaignDeny struct {
	db     *bun.DB
	reason string
}

func (c *campaignDeny) CustomIDPrefix() string {
	return messages.CampaignDenyPrefix
}

func (c *campaignDeny) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, userID, ok := parseApprovalInteraction(s, i)
	if !ok {
		return
	}

	if !checkModAuth(c.db, s, i, userID) {
		return
	}

	// Open a modal to collect the denial reason.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.CampaignDenyModalPrefix + ":" + campaignID,
			Title:    messages.CampaignDenyModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.CampaignDenyReasonFieldID,
						Label:       messages.CampaignDenyReasonLabel,
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Placeholder: messages.CampaignDenyReasonPlaceholder,
					},
				}},
			},
		},
	})
}

// parseApprovalInteraction extracts the campaign ID and user ID from an approval button click.
// Note: In DMs, i.Member is nil. i.User is used instead.
func parseApprovalInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) (campaignID, userID string, ok bool) {
	parts := strings.SplitN(i.MessageComponentData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return "", "", false
	}

	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	return parts[1], userID, true
}

// checkModAuth verifies the user is a mod or admin.
func checkModAuth(database *bun.DB, s *discordgo.Session, i *discordgo.InteractionCreate, userID string) bool {
	ok, err := auth.Authorize(database, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		respondInteraction(s, i, messages.CampaignApproveNotModError)
		return false
	}
	return true
}

// campaignDenyModal handles the modal submission after a mod clicks "Deny" and provides a reason.
// Custom ID format: campaign_deny_modal:<campaignID>
type campaignDenyModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (m *campaignDenyModal) CustomIDPrefix() string {
	return messages.CampaignDenyModalPrefix
}

func (m *campaignDenyModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.ModalSubmitData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[1]

	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	if !checkModAuth(m.db, s, i, userID) {
		return
	}

	var reason string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			if input.CustomID == messages.CampaignDenyReasonFieldID {
				reason = input.Value
			}
		}
	}

	campaign, err := db.GetByID[models.Campaign](m.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignApproveNotFound)
		return
	}

	if err := db.Delete[models.Campaign](m.db, campaignID); err != nil {
		log.Printf("campaign_deny_modal: failed to delete campaign %s: %v", campaignID, err)
		respondInteraction(s, i, messages.CampaignApproveError)
		return
	}

	m.dispatcher.Push(dispatch.DirectMessage{
		ID:      campaignID,
		Sender:  userID,
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.CampaignDeniedDMMessage, campaign.Name, reason),
	})

	respondInteraction(s, i, fmt.Sprintf(messages.CampaignDeniedMessage, campaign.Name))
}
