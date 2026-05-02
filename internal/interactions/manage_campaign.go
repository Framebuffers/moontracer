package interactions

/*
	Managing a campaign is a multi-stage flow. This is the flow for the `/managecampaigns` command.
	Flow:
		1. Authorize: check if the player is registered.
		2. Load all the Player's Campaigns, filter the ones where Role == DM
		3. Renders an ephemeral list with a button per campaign (CustomID: "manage_campaign:<id>")
*/

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/guard"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
manageCampaignMenu provides a model to select options in a menu providing options to manage a Campaign. interaction: `manage_campaign:<id>` [interaction: menu]
 1. Authorize: check if invoker is DM or Mod for that Campaign.
 2. Show action buttons: [Edit, Delete, Ban, Announce, Reschedule]
*/
type manageCampaignMenu struct {
	db *bun.DB
}

func (h *manageCampaignMenu) CustomIDPrefix() string {
	return "manage_campaign"
}

func (h *manageCampaignMenu) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	RenderManageCampaignMenu(s, i, h.db, parts[1])
}

/*
RenderManageCampaignMenu renders the management menu for a campaign.

Used by the manage_campaign handler and back_manage_campaign handler.
*/
func RenderManageCampaignMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	userID := getUserID(i)

	/*
		Check campaign exists before auth. Prevents misleading "not authorized"
		when selecting a deleted campaign from a stale menu.
	*/
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.ManageCampaignNotFound)
		return
	}

	ok, err := auth.Authorize(database, userID, auth.ScopeDM, campaignID)
	if err != nil {
		log.Printf("manage_campaign: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return
	}

	if !requireMutable(s, i, campaign) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Managing **%s**:", campaign.Name),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    messages.ManageEditLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("%s:%s", messages.ManageEditPrefix, campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageDeleteLabel,
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("manage_delete:%s", campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageBanLabel,
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("manage_ban:%s", campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageAnnounceLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("manage_announce:%s", campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageRescheduleLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("manage_reschedule:%s", campaignID),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    messages.ManageSetRoleLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("%s:%s", messages.ManageSetRolePrefix, campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageInviteLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("%s:%s", messages.ManageInvitePrefix, campaignID),
						},
						discordgo.Button{
							Label:    messages.ManageArchiveLabel,
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("%s:%s", messages.ManageArchivePrefix, campaignID),
						},
						discordgo.Button{
							Label:    messages.SetCoverButtonLabel,
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("manage_setcover:%s", campaignID),
						},
						router.BackButton(messages.BackLabel, router.ViewManage),
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
manageCampaignDelete is a model with information to ban a Player. Interaction: `manage_delete:<id>` [delete]

 1. Authorize: invoker must be DM of that Campaign or Mod.
 2. Delete all CampaignMembers from that Campaign, then delete the Campaign itself.
*/
type manageCampaignDelete struct {
	db *bun.DB
}

func (h *manageCampaignDelete) CustomIDPrefix() string {
	return "manage_delete"
}

func (h *manageCampaignDelete) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := loadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !requireMutable(s, i, campaign) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.ManageDeleteConfirm, campaign.Name),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.ManageDeleteConfirmLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.ManageDeleteConfirmID, campaignID),
					},
					discordgo.Button{
						Label:    messages.ManageDeleteCancelLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: router.NavCustomID(router.ViewManageCampaign, campaignID),
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
manageDeleteConfirm handles the "Yes, Delete" confirmation button.

Custom ID format: manage_delete_confirm:<campaignID>
*/
type manageDeleteConfirm struct {
	db *bun.DB
}

func (h *manageDeleteConfirm) CustomIDPrefix() string {
	return messages.ManageDeleteConfirmID
}

func (h *manageDeleteConfirm) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := getUserID(i)

	campaign, ok := loadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !requireMutable(s, i, campaign) {
		return
	}

	ctx := context.Background()
	_, err := h.db.NewDelete().Model((*models.CampaignPlayer)(nil)).
		Where("campaign_id = ?", campaignID).Exec(ctx)
	if err != nil {
		log.Printf("manage_delete_confirm: failed to delete campaign players: %v", err)
		respondInteraction(s, i, messages.ManageDeleteFailure)
		return
	}

	if err := db.Delete[models.Campaign](h.db, campaignID); err != nil {
		log.Printf("manage_delete_confirm: failed to delete campaign: %v", err)
		respondInteraction(s, i, messages.ManageDeleteFailure)
		return
	}

	log.Printf("manage_delete_confirm: %s deleted campaign %s (%s)", userID, campaign.Name, campaignID)
	respondInteraction(s, i, fmt.Sprintf(messages.ManageDeleteSuccess, campaign.Name))
}

/*

truth table: can ban?
							+-----------+-----------+-------+-----------+-----------+-------------------+
							|	admin 	|	mod		|  dm	|	player	|	member	|	unregistered	|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	admin			|	  F		|	  T		|	T	|	  T		|	  T		|		T			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	mod 			|	  F		|	  F		|	T	|	  T		|	  T		|		T			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	dm				|	  F		|	  F		|	F	|	  T*	|	  F		|		F			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	player			|	  F		|	  F		|	F	|	  F		| 	  F		|		F			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	member			|	  F		|	  F		|	F	|	  F		| 	  F		|		F			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+
		|	unregistered	|	  F		|	  F		|	F	|	  F		| 	  F		|		F			|
		--------------------+-----------+-----------+-------+-----------+-----------+-------------------+

		basically:
			- if you have banning permissions, you cannot ban yourself.
			- as said before, permissions cascade.
			- the *only* special case are DMs, where they have a **limited scope** for banning permissions (their Campaign).
			- apart from DMs, anyone below cannot ban anyone else.

*/

/*
manageCampaignBan is a model with information to ban a member from a Campaign. Interaction: `manage_ban:<id>` [ban select]
 1. Authorize: invoker must be a DM of that Campaign or Mod.
 2. Load active members (excluding the DM or already banned users)
 3. Show a select menu dropdown.
*/
type manageCampaignBan struct {
	db *bun.DB
}

func (h *manageCampaignBan) CustomIDPrefix() string {
	return "manage_ban"
}

func (h *manageCampaignBan) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID)
	if err != nil {
		log.Printf("manage_ban: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.ManageCampaignNotFound)
		return
	}

	if !requireMutable(s, i, campaign) {
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("manage_ban: failed to load players: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}

	var options []discordgo.SelectMenuOption
	for _, p := range players {
		if p.Role == models.RoleDM || p.Status == models.StatusBanned || p.PlayerID == userID {
			continue
		}
		label := p.PlayerID
		if p.Player != nil {
			label = p.Player.ID // Discord user ID — shown as fallback
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: label,
			Value: fmt.Sprintf("%s:%s", campaignID, p.PlayerID),
		})
	}

	if len(options) == 0 {
		respondInteraction(s, i, messages.ManageBanNoMembers)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.ManageSelectMember, campaign.Name),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "manage_ban_select",
							Placeholder: "Select a member...",
							Options:     options,
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
manageCampaignBanSelect is a model that returns information to execute a ban action. Interaction: `[manage_ban_select]`

 1. Get `campaignID:playerID` from manageCampaignBan.

 2. Split values into two separate values.

 3. Authorize

 4. Call `SetCampaignPlayerStatus` and set it to models.StatusBanned.

    Notes:

    - Applies only to the campaign being passed. Scope is limited to Campaign only.

    - Invoker must be a DM of this Campaign or have a heavier role (Mod or Admin).
*/
type manageCampaignBanSelect struct {
	db *bun.DB
}

func (h *manageCampaignBanSelect) CustomIDPrefix() string {
	return "manage_ban_select"
}

func (h *manageCampaignBanSelect) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}

	parts := strings.SplitN(values[0], ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[0]
	targetID := parts[1]

	ok, err := auth.Authorize(h.db, invokerID, auth.ScopeDM, campaignID)
	if err != nil {
		log.Printf("manage_ban_select: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return
	}

	err = models.SetCampaignPlayerStatus(h.db, targetID, campaignID, models.StatusBanned)
	if err != nil {
		log.Printf("manage_ban_select: failed to set status to banned: %v", err)
		respondInteraction(s, i, fmt.Sprintf("Could not ban %s from %s.", targetID, campaignID))
		return
	}

	// Remove the campaign's linked Discord role if one exists.
	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err == nil && campaign.RoleID != "" {
		if err := guard.GuildMemberRoleRemove(s, i.GuildID, targetID, campaign.RoleID); err != nil {
			log.Printf("manage_ban_select: failed to remove role %s from %s: %v", campaign.RoleID, targetID, err)
		}
	}

	log.Printf("manage_ban_select: banned successfully. target: %s, campaign: %s", targetID, campaignID)
	respondInteraction(s, i, fmt.Sprintf("%s has been banned from Campaign %s.", targetID, campaignID))
}
