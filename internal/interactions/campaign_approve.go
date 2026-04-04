package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// campaignApprove handles the "Approve" button click on a campaign approval DM.
// Custom ID format: campaign_approve:<campaignID>
type campaignApprove struct {
	db *bun.DB
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

	respondInteraction(s, i, fmt.Sprintf(messages.CampaignApprovedMessage, campaign.Name))
}

// campaignDeny handles the "Deny" button click on a campaign approval DM.
// Custom ID format: campaign_deny:<campaignID>
type campaignDeny struct {
	db *bun.DB
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

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignApproveNotFound)
		return
	}

	if err := db.Delete[models.Campaign](c.db, campaignID); err != nil {
		log.Printf("campaign_deny: failed to delete campaign %s: %v", campaignID, err)
		respondInteraction(s, i, messages.CampaignApproveError)
		return
	}

	respondInteraction(s, i, fmt.Sprintf(messages.CampaignDeniedMessage, campaign.Name))
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
