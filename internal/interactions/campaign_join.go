package interactions

import (
	"fmt"
	"log"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
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
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *campaignJoin) CustomIDPrefix() string {
	return "campaign_join"
}

func (h *campaignJoin) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// split the custom ID in two: id and params
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	tag := parts[1]
	userID := i.Member.User.ID

	// Is the player registered and not globally banned?
	ok, err := auth.Authorize(h.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("campaign_join: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		helpers.RespondNotRegistered(s, i)
		return
	}

	// does the campaign exist and is it active?
	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !helpers.IsCampaignMutable(s, i, campaign) {
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
		helpers.RespondUpdateTerminal(s, i, messages.CampaignClosedMessage)
		return
	}

	// is the player already a member?
	players, err := models.GetCampaignPlayers(h.db, campaign.ID)
	if err != nil {
		log.Printf("campaign_join: %s: %v", messages.PlayerFetchErrorMessage, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	for _, p := range players {
		if p.PlayerID == userID {
			if p.Status == models.StatusBanned {
				helpers.RespondUpdateTerminal(s, i, messages.PlayerBannedMessage)
				return
			}
			helpers.RespondUpdateTerminal(s, i, messages.PlayerAlreadyOnCampaignMessage)
			return
		}
	}

	/*
		Roster cap.

		Westmarches store math.MaxInt32, so this never trips for them;
		their tripwire is SessionCapacity, evaluated below after admission.
	*/
	activePlayerCount := 0
	for _, p := range players {
		if p.Status == models.StatusActive {
			activePlayerCount++
		}
	}
	if !campaign.IsWestmarch && campaign.Slots > 0 && activePlayerCount >= campaign.Slots {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignFullMessage)
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
		helpers.RespondUpdateTerminal(s, i, messages.PlayerFailedToJoinMessage)
		return
	}
	newActiveCount := activePlayerCount + 1

	// if the campaign has a role set already, assign it to the player.
	if campaign.RoleID != "" {
		if err := guard.GuildMemberRoleAdd(s, i.GuildID, userID, campaign.RoleID); err != nil {
			log.Printf("campaign_join: failed to assign role %s to %s: %v", campaign.RoleID, userID, err)
		}
	}

	// Auto-close when the last slot fills.
	if !campaign.IsWestmarch && campaign.Slots > 0 && newActiveCount >= campaign.Slots && campaign.IsOpen {
		campaign.IsOpen = false
		if err := db.Update(h.db, campaign); err != nil {
			log.Printf("campaign_join: auto-close failed for campaign %s: %v", campaign.ID, err)
		} else {
			h.dispatcher.Push(dispatch.DirectMessage{
				ID:      uuid.NewString(),
				Target:  campaign.DungeonMaster,
				Content: fmt.Sprintf(messages.CampaignAutoClosedDM, campaign.Name, campaign.Slots),
			})
		}
	}

	// Westmarch tripwire: admit, then alert if the limit is passed.
	if campaign.IsWestmarch && campaign.SessionCapacity > 0 && newActiveCount > campaign.SessionCapacity {
		h.dispatcher.Push(dispatch.DirectMessage{
			ID:     uuid.NewString(),
			Sender: userID,
			Target: campaign.DungeonMaster,
			Content: fmt.Sprintf(messages.WestmarchOverCapacityDMAlert,
				userID, campaign.Name, newActiveCount, campaign.SessionCapacity),
		})
		helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.WestmarchOverCapacityPlayerNotice,
			campaign.Name, campaign.SessionCapacity))
		return
	}

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.PlayerJoinedCampaignMessage, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.BrowseMoreLabel, discordgo.SecondaryButton, router.ViewCampaignsBrowse),
			router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
		}},
	})
}
