package interactions

/*
	Admin Campaigns handler for the /admin hub.

	Flow:
		1. Staff clicks "Active Campaigns" on the /admin hub.
		2. Auth: ScopeMod.
		3. Load ALL campaigns (approved + unapproved, not archived).
		4. Render a select menu or paginated list with campaign name, DM, status flags.
		5. Selecting a campaign could show admin-level detail or actions.
		6. Back button returns to /admin hub.
*/

import (
	"moontracer/internal/interactions/helpers"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type adminCampaignsHandler struct {
	db *bun.DB
}

func (h *adminCampaignsHandler) CustomIDPrefix() string {
	return messages.AdminCampaignsPrefix
}

func (h *adminCampaignsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		helpers.Respond(s, i, messages.CampaignDBNotStaff)
		return
	}

	campaigns, err := db.GetAll[models.Campaign](h.db)
	if err != nil {
		log.Printf("admin_campaigns: failed to load campaigns: %v", err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	var filtered []models.Campaign
	for _, c := range campaigns {
		if !c.IsArchived {
			filtered = append(filtered, c)
		}
	}

	if len(filtered) == 0 {
		helpers.Respond(s, i, messages.AdminCampaignsNone)
		return
	}

	var lines []string
	for _, c := range filtered {
		lines = append(lines, fmt.Sprintf("• **%s** — DM <@%s> [%s]",
			c.Name, c.DungeonMaster, messages.BuildFlags(c)))
	}

	overview := strings.Join(lines, "\n")
	if len(overview) > 4000 {
		overview = overview[:4000] + "\n... (truncated)"
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s (%d)", messages.AdminCampaignsHeader, len(filtered)),
		Description: overview,
	}

	var options []discordgo.SelectMenuOption
	for idx, c := range filtered {
		if idx >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       c.Name,
			Description: fmt.Sprintf("DM: %s - %s", c.DungeonMaster, messages.BuildFlags(c)),
			Value:       c.ID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:  []*discordgo.MessageEmbed{embed},
			Content: "",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    messages.AdminCampaignSelectPrefix,
						Placeholder: "Pick a campaign for details...",
						Options:     options,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.BackButton(messages.BackLabel, router.ViewAdmin),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
adminCampaignSelectHandler renders an admin-level detail panel when the user
picks a campaign from the dropdown rendered by adminCampaignsHandler.

CustomID: admin_campaign_select (the value carries the campaign ID).
*/
type adminCampaignSelectHandler struct {
	db *bun.DB
}

func (h *adminCampaignSelectHandler) CustomIDPrefix() string {
	return messages.AdminCampaignSelectPrefix
}

func (h *adminCampaignSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		helpers.Respond(s, i, messages.CampaignDBNotStaff)
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.Respond(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := values[0]

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.Respond(s, i, messages.ManageCampaignNotFound)
		return
	}

	roleLine := "_no role linked_"
	if campaign.RoleID != "" {
		roleLine = fmt.Sprintf("<@&%s>", campaign.RoleID)
	}

	scheduleLine := fmt.Sprintf("%s %s UTC (%.1fh, %s)",
		campaign.Schedule.DayName(),
		campaign.Schedule.StartTime,
		campaign.Schedule.DurationHours,
		campaign.Schedule.Frequency)

	body := fmt.Sprintf(
		"**Tag:** `%s`\n**DM:** <@%s>\n**Status:** %s\n**Slots:** %d\n**Role:** %s\n**Schedule:** %s",
		campaign.Tag,
		campaign.DungeonMaster,
		messages.BuildFlags(*campaign),
		campaign.Slots,
		roleLine,
		scheduleLine,
	)

	if campaign.Description != "" {
		body += "\n\n**Description:**\n" + campaign.Description
	}

	embed := &discordgo.MessageEmbed{
		Title:       campaign.Name,
		Description: body,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: "",
			Embeds:  []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.BackButton(messages.BackLabel, router.ViewAdmin),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
