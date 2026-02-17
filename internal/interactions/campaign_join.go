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

type campaignJoin struct {
	db *bun.DB
}

func (h *campaignJoin) CustomIDPrefix() string {
	return "campaign_join"
}

func (h *campaignJoin) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// split the custom ID in two: id and params
	parts := strings.SplitN(i.MessageComponentData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[1]
	userID := i.Member.User.ID

	// is the player registered?
	_, err := db.GetByID[models.Player](h.db, userID)
	if err != nil {
		respondInteraction(s, i, messages.NotRegisteredMessage)
		return
	}

	// does the campaign exist and is it active?
	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}
	if !campaign.IsOpen {
		respondInteraction(s, i, messages.CampaignClosedMessage)
		return
	}

	// is the player already a member?
	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("%s: %v", messages.PlayerFetchErrorMessage, err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	for _, p := range players {
		if p.PlayerID == userID {
			if p.Status == models.StatusBanned {
				respondInteraction(s, i, messages.PlayerBannedMessage)
				return
			}
			respondInteraction(s, i, messages.PlayerAlreadyOnCampaignMessage)
			return
		}
	}

	// are there any slots available?
	activePlayerCount := 0
	for _, p := range players {
		if p.Status == models.StatusActive {
			activePlayerCount++
		}
	}
	if activePlayerCount >= campaign.Slots {
		respondInteraction(s, i, messages.CampaignFullMessage)
		return
	}

	cp := &models.CampaignPlayer{
		PlayerID:   userID,
		CampaignID: campaignID,
		Role:       models.RolePlayer,
		Status:     models.StatusActive,
	}
	if err := db.Insert(h.db, cp); err != nil {
		log.Printf("%s %v", messages.InsertPlayerErrorMessage, err)
		respondInteraction(s, i, messages.PlayerFailedToJoinMessage)
		return
	}

	respondInteraction(s, i, fmt.Sprintf("%s **%s**!", messages.PlayerJoinedCampaignMessage, campaignID))
}
