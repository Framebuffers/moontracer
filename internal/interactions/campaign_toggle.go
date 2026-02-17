package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type campaignToggle struct {
	db *bun.DB
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
	campaignID := parts[1]
	userID := i.Member.User.ID

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if campaign.DungeonMaster != userID {
		respondInteraction(s, i, messages.MasterCanToggleStatusErrorMessage)
		return
	}

	campaign.IsOpen = !campaign.IsOpen
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("%s %v", messages.CampaignUpdateErrorMessage, err)
		respondInteraction(s, i, messages.CampaignUpdateErrorMessage)
		return
	}

	status := "closed"
	if campaign.IsOpen {
		status = "open"
	}
	respondInteraction(s, i, fmt.Sprintf("%s **%s** is now **%s**.", messages.CampaignStatusMessage, campaignID, status))
}
