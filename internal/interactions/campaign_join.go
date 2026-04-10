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
		2. User clicks the "Join Campaign" button, triggering `campaign_join:X`.
		3. `campaignJoin` validates: user is registered, campaign exists & is open, user not already in it.
		4. Inserts CampaignPlayer record with StatusPending (awaiting DM approval).
		5. Responds to user ephemerally: "Your join request has been sent to the DM for approval."
*/

// campaignJoin handles when a player clicks "Join Campaign" on an open campaign.
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
	tag := parts[1]
	userID := i.Member.User.ID

	// Is the player registered and not globally banned?
	ok, err := auth.Authorize(h.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("campaign_join: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.NotRegisteredMessage)
		return
	}

	// does the campaign exist and is it active?
	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.CanMutate() {
		respondInteraction(s, i, messages.CampaignArchivedMessage)
		return
	}

	// Check if the player has the campaign's linked Discord role.
	hasLinkedRole := false
	if campaign.RoleID != "" {
		for _, roleID := range i.Member.Roles {
			if roleID == campaign.RoleID {
				hasLinkedRole = true
				break
			}
		}
	}

	// Campaign must be open OR the player must have the linked role.
	if !campaign.IsOpen && !hasLinkedRole {
		respondInteraction(s, i, messages.CampaignClosedMessage)
		return
	}

	// is the player already a member?
	players, err := models.GetCampaignPlayers(h.db, campaign.ID)
	if err != nil {
		log.Printf("campaign_join: %s: %v", messages.PlayerFetchErrorMessage, err)
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
		CampaignID: campaign.ID,
		Role:       models.RolePlayer,
		Status:     models.StatusActive,
	}
	if err := db.Insert(h.db, cp); err != nil {
		log.Printf("campaign_join: %s: %v", messages.InsertPlayerErrorMessage, err)
		respondInteraction(s, i, messages.PlayerFailedToJoinMessage)
		return
	}

	// if the campaign has a role set already, assign it to the player.
	if campaign.RoleID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, userID, campaign.RoleID); err != nil {
			log.Printf("campaign_join: failed to assign role %s to %s: %v", campaign.RoleID, userID, err)
		}
	}

	respondInteraction(s, i, fmt.Sprintf("%s **%s**!", messages.PlayerJoinedCampaignMessage, campaign.Name))
}
