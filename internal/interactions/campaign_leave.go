package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. User clicks `/campaign tag:X` to view campaign details.
		2. User clicks the "Leave Campaign" button, triggering `campaign_leave:X`.
		3. `campaignLeave` validates: campaign exists, user is active member, user is not the DM.
		4. Deletes the CampaignPlayer record for that user.
		5. Responds to user ephemerally: "You have left campaign <Name>".
*/

// campaignLeave handles when a player clicks "Leave Campaign" to remove themselves.
type campaignLeave struct {
	db *bun.DB
}

func (h *campaignLeave) CustomIDPrefix() string {
	return "campaign_leave"
}

func (h *campaignLeave) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.MessageComponentData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	tag := parts[1]
	userID := i.Member.User.ID

	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	// DMs cannot leave their own campaign.
	isDM, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("campaign_leave: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if isDM {
		respondInteraction(s, i, messages.MasterIsLeavingCampaignErrorMessage)
		return
	}

	if err := models.RemoveCampaignPlayer(h.db, userID, campaign.ID); err != nil {
		log.Printf("campaign_leave: %s: %v", messages.LeavingCampaignErrorMessage, err)
		respondInteraction(s, i, messages.FailedToLeaveCampaignErrorMessage)
		return
	}

	// Remove the campaign's linked Discord role if one exists.
	if campaign.RoleID != "" {
		if err := guard.GuildMemberRoleRemove(s, i.GuildID, userID, campaign.RoleID); err != nil {
			log.Printf("campaign_leave: failed to remove role %s from %s: %v", campaign.RoleID, userID, err)
		}
	}

	respondInteraction(s, i, fmt.Sprintf("%s **%s**.", messages.PlayerLeftCampaignMessage, campaign.Name))
}
