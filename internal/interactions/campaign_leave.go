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

	// DMs cannot leave their own campaign
	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}
	if campaign.DungeonMaster == userID {
		respondInteraction(s, i, messages.MasterIsLeavingCampaignErrorMessage)
		return
	}

	if err := models.RemoveCampaignPlayer(h.db, userID, campaign.ID); err != nil {
		log.Printf("%s %v", messages.LeavingCampaignErrorMessage, err)
		respondInteraction(s, i, messages.FailedToLeaveCampaignErrorMessage)
		return
	}

	respondInteraction(s, i, fmt.Sprintf("%s **%s**.", messages.PlayerLeftCampaignMessage, campaign.Name))
}
