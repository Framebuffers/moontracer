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

/*
	Flow:
		1. User clicks `/campaign tag:X` to view campaign details.
		2. DM clicks "Set as Open/Closed Campaign" button, triggering `campaign_toggle:X`.
		3. `campaignToggle` validates: campaign exists, user is the DM.
		4. Toggles campaign.IsOpen (open → closed, closed → open).
		5. Updates campaign in DB.
		6. Responds to DM ephemerally with the new status.
*/

// campaignToggle handles when a DM clicks to toggle a campaign between open/closed.
type campaignToggle struct {
	db            *bun.DB
	guildID       string
	adminRoleName string
}

func (h *campaignToggle) CustomIDPrefix() string {
	return "campaign_toggle"
}

func (h *campaignToggle) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	ok, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("campaign_toggle: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.MasterCanToggleStatusErrorMessage)
		return
	}

	campaign.IsOpen = !campaign.IsOpen
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("campaign_toggle: %s: %v", messages.CampaignUpdateErrorMessage, err)
		respondInteraction(s, i, messages.CampaignUpdateErrorMessage)
		return
	}

	status := "closed"
	if campaign.IsOpen {
		status = "open"
	}
	respondInteraction(s, i, fmt.Sprintf("%s **%s** is now **%s**.", messages.CampaignStatusMessage, campaign.Name, status))
}
