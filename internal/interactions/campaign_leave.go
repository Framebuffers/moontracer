package interactions

import (
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
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
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	tag := parts[1]
	userID := i.Member.User.ID

	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	// DMs cannot leave their own campaign.
	isDM, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("campaign_leave: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if isDM {
		helpers.RespondUpdateTerminal(s, i, messages.MasterIsLeavingCampaignErrorMessage)
		return
	}

	if err := models.RemoveCampaignPlayer(h.db, userID, campaign.ID); err != nil {
		log.Printf("campaign_leave: %s: %v", messages.LeavingCampaignErrorMessage, err)
		helpers.RespondUpdateTerminal(s, i, messages.FailedToLeaveCampaignErrorMessage)
		return
	}

	// Remove the campaign's linked Discord role if one exists.
	if campaign.RoleID != "" {
		if err := guard.GuildMemberRoleRemove(s, i.GuildID, userID, campaign.RoleID); err != nil {
			log.Printf("campaign_leave: failed to remove role %s from %s: %v", campaign.RoleID, userID, err)
		}
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.PlayerLeftCampaignMessage, campaign.Name))
}
