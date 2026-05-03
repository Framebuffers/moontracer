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
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
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

	const nameW, statusW, dmW = 20, 10, 18
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-*s %-*s %-4s %-*s %s\n",
		nameW, "CAMPAIGN", statusW, "STATUS", "OPEN", dmW, "DM ID", "NEXT SESSION"))
	sb.WriteString(strings.Repeat("─", nameW+statusW+dmW+30) + "\n")
	for _, c := range filtered {
		name := c.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		status := messages.BuildFlags(c)
		if len(status) > statusW {
			status = status[:statusW-1] + "…"
		}
		open := "no"
		if c.IsOpen {
			open = "yes"
		}
		dmTail := c.DungeonMaster
		if len(dmTail) > dmW {
			dmTail = "…" + dmTail[len(dmTail)-(dmW-1):]
		}
		next := "(not set)"
		if !c.Schedule.NextSession.IsZero() {
			next = c.Schedule.NextSession.Format("2006-01-02")
		}
		sb.WriteString(fmt.Sprintf("%-*s %-*s %-4s %-*s %s\n",
			nameW, name, statusW, status, open, dmW, dmTail, next))
	}
	block := "```\n" + sb.String() + "```"
	if len(block) > 1900 {
		block = block[:1896] + "…```"
	}

	var options []discordgo.SelectMenuOption
	for idx, c := range filtered {
		if idx >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       c.Name,
			Description: fmt.Sprintf("DM: %s · %s", c.DungeonMaster, messages.BuildFlags(c)),
			Value:       c.ID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: block,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    messages.AdminCampaignSelectPrefix,
						Placeholder: messages.AdminCampaignSelectPlaceholder,
						Options:     options,
					},
				}},
				helpers.BackRow(router.ViewAdmin),
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
					discordgo.Button{
						Label: messages.AdminContactDMLabel,
						Style: discordgo.LinkButton,
						URL:   fmt.Sprintf("discord://-/users/%s", campaign.DungeonMaster),
					},
					router.BackButton(messages.BackLabel, router.ViewAdmin),
					router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
