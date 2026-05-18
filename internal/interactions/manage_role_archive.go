package interactions

/*
	Handlers for the Set Role and Archive buttons on the manage campaign menu.

	Set Role flow:
		1. Button click (manage_role:<campaignID>): open a modal asking for a role name.
		2. Modal submit (modal_manage_role:<campaignID>): find-or-create Discord role, link to campaign.

	Archive flow:
		1. Button click (manage_archive:<campaignID>): show confirmation message.
		2. Confirm (manage_archive_confirm:<campaignID>): archive via ArchiveCampaign + audit.
		3. Cancel (manage_archive_cancel:<campaignID>): return to manage menu.
*/

import (
	"fmt"
	"log"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/scheduler"
)

/*
	Set Role:
	Button to Modal
*/

type manageSetRole struct {
	db *bun.DB
}

func (h *manageSetRole) CustomIDPrefix() string {
	return messages.ManageSetRolePrefix
}

func (h *manageSetRole) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.ManageSetRoleModalID, campaignID),
			Title:    messages.ManageSetRoleModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "role_name",
						Label:       messages.ManageSetRoleFieldLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						MaxLength:   100,
						Placeholder: "e.g. Curse of Strahd",
					},
				}},
			},
		},
	})
}

/*
	Set Role:
	Modal Submit
*/

type manageSetRoleModal struct {
	db *bun.DB
}

func (h *manageSetRoleModal) CustomIDPrefix() string {
	return messages.ManageSetRoleModalID
}

func (h *manageSetRoleModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	roleName := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	if roleName == "" {
		helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
		return
	}

	// discord role: find or create
	var roleID string
	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		log.Printf("manage_role: failed to fetch guild roles: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
		return
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, roleName) {
			roleID = role.ID
			break
		}
	}

	if roleID == "" {
		role, err := guard.GuildRoleCreate(s, i.GuildID, &discordgo.RoleParams{
			Name: roleName,
		})
		if err != nil {
			log.Printf("manage_role: failed to create role: %v", err)
			helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
			return
		}
		roleID = role.ID
	}

	campaign.RoleID = roleID
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_role: failed to update campaign: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
		return
	}

	log.Printf("manage_role: %s linked role %s (%s) to campaign %s", userID, roleName, roleID, campaign.Name)
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageSetRoleSuccess, roleName, campaign.Name))
}

/*
	Archive flow:
	Button for confirmation.
*/

type manageArchive struct {
	db *bun.DB
}

func (h *manageArchive) CustomIDPrefix() string {
	return messages.ManageArchivePrefix
}

func (h *manageArchive) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			Content: fmt.Sprintf(messages.ManageArchiveConfirm, campaign.Name),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.ManageArchiveConfirmLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.ManageArchiveConfirmID, campaignID),
					},
					discordgo.Button{
						Label:    messages.ManageArchiveCancelLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: router.NavCustomID(router.ViewManageDanger, campaignID),
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
	Archive flow:
	Confirm button
*/

type manageArchiveConfirm struct {
	db    *bun.DB
	sched *scheduler.Scheduler
}

func (h *manageArchiveConfirm) CustomIDPrefix() string {
	return messages.ManageArchiveConfirmID
}

func (h *manageArchiveConfirm) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	if err := commands.ArchiveCampaign(h.db, campaign, messages.AbandonReasonDM); err != nil {
		log.Printf("manage_archive: failed to archive campaign %s: %v", campaign.ID, err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageArchiveFailed)
		return
	}
	h.sched.Cancel(i.GuildID, campaign.ID)
	RetireChannel(s, i.GuildID, campaign)

	if err := models.InsertAuditEntry(h.db, userID, userID, models.AuditCampaignArchive, fmt.Sprintf("archived campaign %s (%s) via manage menu", campaign.Name, campaign.Tag)); err != nil {
		log.Printf("manage_archive: failed to write audit entry: %v", err)
	}

	log.Printf("manage_archive: %s archived campaign %s (%s)", userID, campaign.Name, campaign.ID)
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageArchiveSuccess, campaign.Name), nil, []discordgo.MessageComponent{
		helpers.BackRow(router.ViewManage),
	})
}
