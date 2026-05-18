package interactions

/*
	Player invitation flow.

	Flow:
		1. DM clicks "Invite Player" on manage menu → user select dropdown.
		2. DM picks a user → validate (registered, not in campaign, campaign not full).
		3. Create CampaignPlayer with StatusPending, send invitation DM with Accept/Decline.
		4. Target clicks Accept → status becomes active, role assigned, DM message updated.
		5. Target clicks Decline → CampaignPlayer removed, DM message updated.

	CustomID formats:
		manage_invite:<campaignID>
		manage_invite_select:<campaignID>
		campaign_invite_accept:<guildID>:<campaignID>
		campaign_invite_decline:<guildID>:<campaignID>
*/

import (
	"fmt"
	"log"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// manageCampaignInvite shows a user select menu for the DM to pick a player to invite.
type manageCampaignInvite struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *manageCampaignInvite) CustomIDPrefix() string {
	return messages.ManageInvitePrefix
}

func (h *manageCampaignInvite) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.ManageInviteSelectPrompt, campaign.Name),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.UserSelectMenu,
						CustomID:    fmt.Sprintf("%s:%s", messages.ManageInviteSelectPrefix, campaignID),
						Placeholder: "Select a player...",
					},
				}},
				helpers.BackRow(router.ViewManageCampaign, campaignID),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// manageCampaignInviteSelect handles the user selection from the invite dropdown.
type manageCampaignInviteSelect struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *manageCampaignInviteSelect) CustomIDPrefix() string {
	return messages.ManageInviteSelectPrefix
}

func (h *manageCampaignInviteSelect) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	targetID := values[0]

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	/*
		Guard clauses:
			1. Target must be registered.
	*/
	if _, err := db.GetByID[models.Player](h.db, targetID); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.AddPlayerTargetNotRegistered)
		return
	}

	// 		2. Target must not already be in the campaign.
	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("campaign_invite_select: failed to load players: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	activeCount := 0
	for _, p := range players {
		if p.PlayerID == targetID {
			helpers.RespondUpdateTerminal(s, i, messages.AddPlayerAlreadyInCampaign)
			return
		}
		if p.Status == models.StatusActive {
			activeCount++
		}
	}

	// 		3. Campaign must not be full.
	if campaign.Slots > 0 && activeCount >= campaign.Slots && !campaign.CanOverflow {
		helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.InviteCampaignFull, campaign.Name))
		return
	}

	// 		4. Create pending membership.
	cp := &models.CampaignPlayer{
		PlayerID:   targetID,
		CampaignID: campaignID,
		Role:       models.RolePlayer,
		Status:     models.StatusPending,
	}
	if err := db.Insert(h.db, cp); err != nil {
		log.Printf("campaign_invite_select: failed to insert campaign player: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	/*
		Send invitation DM with accept/decline buttons.
	*/
	guildID := i.GuildID
	h.dispatcher.Push(dispatch.DirectMessage{
		ID:      uuid.NewString(),
		Sender:  userID,
		Target:  targetID,
		Content: fmt.Sprintf(messages.InviteDMMessage, campaign.Name, userID),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Accept",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("%s:%s:%s", messages.InviteAcceptPrefix, guildID, campaignID),
				},
				discordgo.Button{
					Label:    "Decline",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("%s:%s:%s", messages.InviteDeclinePrefix, guildID, campaignID),
				},
			}},
		},
	})

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.InviteSentMessage, targetID, campaign.Name), nil, []discordgo.MessageComponent{
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

// campaignInviteAccept handles the "Accept" button on an invitation DM.
type campaignInviteAccept struct {
	db *bun.DB
}

func (h *campaignInviteAccept) CustomIDPrefix() string {
	return messages.InviteAcceptPrefix
}

func (h *campaignInviteAccept) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// CustomID: campaign_invite_accept:<guildID>:<campaignID>
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	guildID := parts[1]
	campaignID := parts[2]

	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	/*
		Guard clause:
			1. Verify pending invitation exists.
	*/
	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("campaign_invite_accept: failed to load players: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	found := false
	for _, p := range players {
		if p.PlayerID == userID && p.Status == models.StatusPending {
			found = true
			break
		}
	}
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    messages.InviteAlreadyProcessed,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// 		2. Activate membership.
	if err := models.SetCampaignPlayerStatus(h.db, userID, campaignID, models.StatusActive); err != nil {
		log.Printf("campaign_invite_accept: failed to set status: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// 		3. Assign campaign role if one exists.
	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err == nil && campaign.RoleID != "" {
		if err := guard.GuildMemberRoleAdd(s, guildID, userID, campaign.RoleID); err != nil {
			log.Printf("campaign_invite_accept: failed to add role %s to %s: %v", campaign.RoleID, userID, err)
		}
	}

	campaignName := campaignID
	if campaign != nil {
		campaignName = campaign.Name
	}

	log.Printf("campaign_invite_accept: %s accepted invitation to %s", userID, campaignID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf(messages.InviteAcceptedDMUpdate, campaignName),
			Components: []discordgo.MessageComponent{},
		},
	})
}

// campaignInviteDecline handles the "Decline" button on an invitation DM.
type campaignInviteDecline struct {
	db *bun.DB
}

func (h *campaignInviteDecline) CustomIDPrefix() string {
	return messages.InviteDeclinePrefix
}

func (h *campaignInviteDecline) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// CustomID: campaign_invite_decline:<guildID>:<campaignID>
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[2]

	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil {
		userID = i.Member.User.ID
	}

	/*
		Guard clause:
			1. Verify pending invitation exists.
	*/
	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("campaign_invite_decline: failed to load players: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	found := false
	for _, p := range players {
		if p.PlayerID == userID && p.Status == models.StatusPending {
			found = true
			break
		}
	}
	if !found {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    messages.InviteAlreadyProcessed,
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// 		2. Remove the pending membership.
	if err := models.RemoveCampaignPlayer(h.db, userID, campaignID); err != nil {
		log.Printf("campaign_invite_decline: failed to remove player: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	campaignName := campaignID
	if err == nil {
		campaignName = campaign.Name
	}

	log.Printf("campaign_invite_decline: %s declined invitation to %s", userID, campaignID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf(messages.InviteDeclinedDMUpdate, campaignName),
			Components: []discordgo.MessageComponent{},
		},
	})
}
