package interactions

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. User clicks `/campaign tag:X` to view campaign details.
		2. DM clicks "Set as Open/Closed Campaign" button, triggering `campaign_toggle:X`.
		3. `campaignToggle` validates: campaign exists, user is the DM.
		4. Toggles campaign.IsOpen (open -> closed, closed -> open).
		5. Updates campaign in DB.
		6. Responds to DM ephemerally with the new status.
*/

// campaignToggle handles when a DM clicks to toggle a campaign between open/closed.
type campaignToggle struct {
	db *bun.DB
}

func (h *campaignToggle) CustomIDPrefix() string {
	return "campaign_toggle"
}

func (h *campaignToggle) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	ok, err = auth.Authorize(h.db, userID, auth.ScopeDM, campaign.ID)
	if err != nil {
		log.Printf("campaign_toggle: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.MasterCanToggleStatusErrorMessage)
		return
	}

	campaign.IsOpen = !campaign.IsOpen
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("campaign_toggle: %s: %v", messages.CampaignUpdateErrorMessage, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignUpdateErrorMessage)
		return
	}

	go func() {
		if err := helpers.UpdateBillboard(s, h.db, campaign); err != nil {
			log.Printf("campaign_toggle: billboard update for %s: %v", campaign.ID, err)
		}
	}()

	status := "closed"
	if campaign.IsOpen {
		status = "open"
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.CampaignStatusMessage, campaign.Name, status))
}
