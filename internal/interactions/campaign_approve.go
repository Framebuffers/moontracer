package interactions

import (
	"context"
	"fmt"
	"log"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow (Approve):
		1. Mod receives a DM with Approve/Deny buttons for a pending campaign.
		2. Mod clicks "Approve", triggering `campaign_approve:<campaignID>`.
		3. `parseApprovalInteraction` extracts the campaign ID and the mod's user ID from the DM interaction.
		4. `checkModAuth` verifies the user has ScopeMod (mod or admin).
		5. Loads the campaign by ID, sets IsApproved = true, updates the DB.
		6. Sends a DM to the campaign's DM via the dispatcher: "Your campaign has been approved!"
		7. Responds to the mod ephemerally: "Campaign X has been approved."

	Flow (Deny):
		1. Mod clicks "Deny", triggering `campaign_deny:<campaignID>`.
		2. Auth check (same as approve).
		3. Instead of denying immediately, opens a modal asking for a denial reason.
		4. Mod submits the modal, triggering `campaign_deny_modal:<campaignID>`.
		5. `campaignDenyModal` parses the reason, deletes the campaign from the DB.
		6. Sends a DM to the campaign's DM via the dispatcher with the denial reason.
		7. Responds to the mod ephemerally: "Campaign X has been denied and deleted."
*/

/*
campaignApprove handles the "Approve" button click on a campaign approval DM.

Custom ID format: campaign_approve:<campaignID>
*/
type campaignApprove struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (c *campaignApprove) CustomIDPrefix() string {
	return messages.CampaignApprovePrefix
}

/*
HandleComponents processes and renders information that will be presented inside the modal window.

Note:

	createCampaignChannels: creates category, channel and standard Discord Threads.
							errors are logged but non-fatal.
*/
func (c *campaignApprove) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID, campaignID, _, ok := parseApprovalInteraction(s, i)
	if !ok {
		return
	}

	/*
		Auth + DB load are fast; handle errors here before the defer so we can still
		call InteractionRespond without conflicting with a deferred response.
	*/
	campaign, ok := helpers.LoadCampaignAsMod(s, i, c.db, campaignID)
	if !ok {
		return
	}

	campaign.IsApproved = true
	if err := db.Update(c.db, campaign); err != nil {
		log.Printf("campaign_approve: failed to approve campaign %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignApproveError)
		return
	}

	if err := EnsureCampaignRole(s, guildID, campaign); err != nil {
		log.Printf("campaign_approve: role for %s: %v", campaignID, err)
	} else if err := db.Update(c.db, campaign); err != nil {
		log.Printf("campaign_approve: save role for %s: %v", campaignID, err)
	}

	var senderID string
	if i.User != nil {
		senderID = i.User.ID
	} else if i.Member != nil {
		senderID = i.Member.User.ID
	}
	c.dispatcher.Push(dispatch.DirectMessage{
		ID:      campaignID,
		Sender:  senderID,
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.CampaignApprovedDMMessage, campaign.Name),
	})

	content := fmt.Sprintf(messages.CampaignApprovedStatusMessage, campaign.Name)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: []discordgo.MessageComponent{},
		},
	})

	// Channel + thread creation is slow; run after responding so the mod interaction resolves immediately.
	go func() {
		if err := SetupNewChannel(s, guildID, campaign); err != nil {
			log.Printf("campaign_approve: channel setup for %s: %v", campaignID, err)
			return
		}
		if err := db.Update(c.db, campaign); err != nil {
			log.Printf("campaign_approve: save channel IDs for %s: %v", campaignID, err)
		}
	}()
}

/*
campaignDeny handles the "Deny" button click on a campaign approval DM.

Custom ID format: campaign_deny:<campaignID>
*/
type campaignDeny struct {
	db *bun.DB
}

func (c *campaignDeny) CustomIDPrefix() string {
	return messages.CampaignDenyPrefix
}

func (c *campaignDeny) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID, campaignID, userID, ok := parseApprovalInteraction(s, i)
	if !ok {
		return
	}

	if !checkModAuth(c.db, s, i, userID) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.CampaignDenyModalPrefix + ":" + guildID + ":" + campaignID,
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

/*
parseApprovalInteraction extracts the guild ID, campaign ID, and user ID from an approval button click.

Custom ID format: prefix:<guildID>:<campaignID>
Note: In DMs, i.Member is nil. i.User is used instead.
*/
func parseApprovalInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) (guildID, campaignID, userID string, ok bool) {
	parts, valid := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !valid {
		return "", "", "", false
	}

	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	return parts[1], parts[2], userID, true
}

// checkModAuth verifies the user is a mod or admin.
func checkModAuth(database *bun.DB, s *discordgo.Session, i *discordgo.InteractionCreate, userID string) bool {
	ok, err := auth.Authorize(database, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignApproveNotModError)
		return false
	}
	return true
}

/*
campaignDenyModal handles the modal submission after a mod clicks "Deny" and provides a reason.

Custom ID format: campaign_deny_modal:<campaignID>
*/
type campaignDenyModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (m *campaignDenyModal) CustomIDPrefix() string {
	return messages.CampaignDenyModalPrefix
}

func (m *campaignDenyModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// CustomID format: campaign_deny_modal:<guildID>:<campaignID>
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[2]

	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	campaign, ok := helpers.LoadCampaignAsMod(s, i, m.db, campaignID)
	if !ok {
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

	ctx := context.Background()
	if _, err := m.db.NewDelete().Model((*models.CampaignPlayer)(nil)).
		Where("campaign_id = ?", campaignID).Exec(ctx); err != nil {
		log.Printf("campaign_deny_modal: failed to delete campaign players for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignApproveError)
		return
	}

	if err := db.Delete[models.Campaign](m.db, campaignID); err != nil {
		log.Printf("campaign_deny_modal: failed to delete campaign %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignApproveError)
		return
	}

	m.dispatcher.Push(dispatch.DirectMessage{
		ID:      campaignID,
		Sender:  userID,
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.CampaignDeniedDMMessage, campaign.Name, reason),
	})

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.CampaignDeniedStatusMessage, campaign.Name, reason))
}
