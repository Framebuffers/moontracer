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
	"strings"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auditlog"
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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}

	// Pre-fill with the current Discord role name when one is already assigned.
	var currentName string
	if campaign.RoleID != "" {
		roles, err := s.GuildRoles(i.GuildID)
		if err == nil {
			for _, r := range roles {
				if r.ID == campaign.RoleID {
					currentName = r.Name
					break
				}
			}
		}
	}

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
						Value:       currentName,
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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	roleName := strings.TrimSpace(i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	if roleName == "" {
		helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
		return
	}

	if campaign.RoleID != "" {
		// Rename the existing campaign role.
		if _, err := guard.GuildRoleEdit(s, i.GuildID, campaign.RoleID, &discordgo.RoleParams{Name: roleName}); err != nil {
			log.Printf("manage_role: failed to rename role %s: %v", campaign.RoleID, err)
			helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
			return
		}
		log.Printf("manage_role: %s renamed role %s to %q for campaign %s", userID, campaign.RoleID, roleName, campaign.Name)
	} else {
		// No role yet: create one and link it.
		role, err := guard.GuildRoleCreate(s, i.GuildID, &discordgo.RoleParams{Name: roleName})
		if err != nil {
			log.Printf("manage_role: failed to create role: %v", err)
			helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
			return
		}
		campaign.RoleID = role.ID
		if err := db.Update(h.db, campaign); err != nil {
			log.Printf("manage_role: failed to update campaign: %v", err)
			helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
			return
		}
		log.Printf("manage_role: %s created and linked role %s (%s) to campaign %s", userID, roleName, role.ID, campaign.Name)
	}

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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    messages.ManageArchiveInProgress,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err := commands.ArchiveCampaign(h.db, campaign, messages.AbandonReasonDM); err != nil {
		log.Printf("manage_archive: failed to archive campaign %s: %v", campaign.ID, err)
		helpers.EditTerminal(s, i, messages.ManageArchiveFailed)
		return
	}
	h.sched.Cancel(i.GuildID, campaign.ID)
	RetireChannel(s, i.GuildID, campaign)
	MoveToArchivedCategory(h.db, s, campaign)
	DeleteBillboard(s, campaign)

	auditlog.Post(s, h.db, i.GuildID, userID, userID, models.AuditCampaignArchive, fmt.Sprintf("archived campaign %s (%s) via manage menu", campaign.Name, campaign.Tag))

	log.Printf("manage_archive: %s archived campaign %s (%s)", userID, campaign.Name, campaign.ID)
	helpers.EditTerminal(s, i, fmt.Sprintf(messages.ManageArchiveSuccess, campaign.Name))
}

/*
manageLinkRoleHandler shows a Discord role select menu so the DM can link an existing
guild role to their campaign without creating a new one.

CustomID: manage_link_role:<campaignID>
*/
type manageLinkRoleHandler struct {
	db *bun.DB
}

func (h *manageLinkRoleHandler) CustomIDPrefix() string { return messages.ManageLinkRolePrefix }

func (h *manageLinkRoleHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("**Link an existing role to %s**\nPick the Discord role that gates access to this campaign's channel.", campaign.Name),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.RoleSelectMenu,
						CustomID:    fmt.Sprintf("%s:%s", messages.ManageLinkRoleSelectPrefix, campaignID),
						Placeholder: messages.ManageLinkRolePlaceholder,
					},
				}},
				helpers.BackRow(router.ViewManageSettings, campaignID),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
manageLinkRoleSelectHandler saves the selected role to the campaign and applies channel
permissions so the role grants access immediately.

CustomID: manage_link_role_sel:<campaignID>
*/
type manageLinkRoleSelectHandler struct {
	db *bun.DB
}

func (h *manageLinkRoleSelectHandler) CustomIDPrefix() string {
	return messages.ManageLinkRoleSelectPrefix
}

func (h *manageLinkRoleSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	roleID := values[0]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	campaign.RoleID = roleID
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_link_role_sel: update campaign %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageSetRoleFailed)
		return
	}

	if messages.IsSnowflake(campaign.ChannelID) {
		if err := guard.ChannelPermissionSet(s, campaign.ChannelID, roleID,
			discordgo.PermissionOverwriteTypeRole, discordgo.PermissionViewChannel, 0); err != nil {
			log.Printf("manage_link_role_sel: allow role %s on channel %s: %v", roleID, campaign.ChannelID, err)
		}
	}

	_ = strings.TrimSpace // keep strings import used
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageLinkRoleSuccess, campaign.Name))
}
