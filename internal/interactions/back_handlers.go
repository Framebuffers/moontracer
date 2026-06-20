package interactions

/*
	Render functions for the list-style player views.

	RenderMyCampaignsList and RenderManageList are registered with the router
	(see views.go) under ViewMyCampaigns and ViewManage respectively. They are
	invoked both as forward links from /me and as "Back" targets from child
	views that came from one of these lists.

	Flow (RenderMyCampaignsList):
		1. Load CampaignPlayer rows for the invoking user (all roles).
		2. On load failure -> ephemeral error and abort.
		3. Empty result -> empty-state message with only a Back to /me button.
		4. Otherwise build a string select of the user's approved campaigns
		   plus a textual list header, and render via helpers.RespondUpdate
		   (replaces the current ephemeral message in place).

	Flow (RenderManageList):
		1. Load CampaignPlayer rows for the user.
		2. Filter down to entries where Role == DM.
		3. Empty filtered set -> empty-state message with Back only.
		4. Otherwise render a manage select plus a Back + "New Campaign"
		   button row.

	Also exports helpers.GetUserID, a small helper used across the interactions
	package to pull the invoking user's ID out of either the Member (guild
	context) or User (DM) field on an InteractionCreate.

	Historical note:
		Previously this file held one ComponentHandler per
		"back_*" CustomID. Those have been replaced by the view router; the
		rendering bodies live here as package-level functions.
*/

import (
	"fmt"
	"strings"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// RenderMyCampaignsList renders the player's own campaigns as a select menu.
func RenderMyCampaignsList(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	entries, err := models.GetPlayerCampaigns(db, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.MyCampaignsLoadError)
		return
	}

	if len(entries) == 0 {
		helpers.RespondUpdate(s, i, messages.NoCampaignsMessage, nil, []discordgo.MessageComponent{
			helpers.BackRow(router.ViewMe),
		})
		return
	}

	selectMenu := BuildPlayerCampaignSelect(entries, messages.MyCampaignSelectPrefix, messages.MyCampaignsPlaceholder)

	var lines []string
	for _, e := range entries {
		if e.Campaign != nil && e.Campaign.IsApproved {
			lines = append(lines, fmt.Sprintf(messages.MyCampaignListLine, e.Campaign.Name, e.Role, e.Status))
		}
	}
	content := messages.MyCampaignsListHeader + strings.Join(lines, "\n")

	helpers.RespondUpdate(s, i, content, nil, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.PlayerDownloadTokensLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: messages.PlayerDownloadTokensPrefix,
			},
		}},
		helpers.BackRow(router.ViewMe),
	})
}

// RenderManageList renders the campaigns the user DMs as a select menu.
func RenderManageList(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	entries, err := models.GetPlayerCampaigns(db, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	var dmEntries []models.CampaignPlayer
	for _, e := range entries {
		if e.Role == models.RoleDM {
			dmEntries = append(dmEntries, e)
		}
	}

	if len(dmEntries) == 0 {
		helpers.RespondUpdate(s, i, messages.ManageNoDMCampaigns, nil, []discordgo.MessageComponent{
			helpers.BackRow(router.ViewMe),
		})
		return
	}

	// Build select including pending campaigns so DMs can enter and delete them.
	var options []discordgo.SelectMenuOption
	for _, e := range dmEntries {
		if e.Campaign == nil || len(options) >= 25 {
			continue
		}
		label := e.Campaign.Name
		desc := "DM"
		if !e.Campaign.IsApproved {
			desc = "Pending approval- enter to cancel"
		} else if e.Campaign.IsArchived {
			desc = "Archived"
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncate(label, 100),
			Value:       e.CampaignID,
			Description: truncate(desc, 100),
		})
	}
	if len(options) == 0 {
		options = []discordgo.SelectMenuOption{{Label: "No campaigns", Value: "none", Default: true}}
	}
	selectMenu := discordgo.SelectMenu{
		CustomID:    messages.ManageSelectPrefix,
		Placeholder: messages.ManageCampaignsPlaceholder,
		Options:     options,
	}

	var lines []string
	for _, e := range dmEntries {
		if e.Campaign == nil {
			continue
		}
		status := "Active"
		if !e.Campaign.IsApproved {
			status = "Pending"
		} else if e.Campaign.IsArchived {
			status = "Archived"
		}
		lines = append(lines, fmt.Sprintf(messages.ManageCampaignListLine, e.Campaign.Name, status))
	}
	content := messages.ManageCampaignsListHeader + strings.Join(lines, "\n")

	helpers.RespondUpdate(s, i, content, nil, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.NewCampaignLabel,
				Style:    discordgo.SuccessButton,
				CustomID: messages.ManageNewCampaignPrefix,
			},
			router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
		}},
	})
}
