package interactions

/*
	Admin Campaigns handler for the /admin hub.

	Flow:
		1. Staff clicks "Query Campaigns" on the /admin hub.
		2. Auth: ScopeMod.
		3. Load ALL campaigns (approved, pending, archived).
		4. Render a code block with name, status, open/closed, DM name, next session.
		5. Selecting a campaign shows admin-level detail + Contact DM button.
		6. Back button returns to /admin hub.
*/

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
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
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	var filtered []models.Campaign
	err = h.db.NewSelect().
		Model(&filtered).
		WhereAllWithDeleted().
		OrderExpr("(deleted_at IS NOT NULL) ASC, is_archived ASC, is_approved DESC, COALESCE(deleted_at, archived_at, '0001-01-01') DESC").
		Scan(context.Background())
	if err != nil {
		log.Printf("admin_campaigns: failed to load campaigns: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if len(filtered) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.AdminCampaignsNone)
		return
	}

	resolveName := func(userID string) string {
		if m, err := s.State.Member(i.GuildID, userID); err == nil && m.User != nil {
			if m.Nick != "" {
				return m.Nick
			}
			return m.User.Username
		}
		if len(userID) >= 5 {
			return "..." + userID[len(userID)-5:]
		}
		return userID
	}

	const nameW, statusW, dmW = 20, 12, 18
	var sb strings.Builder
	fmt.Fprintf(&sb, "%-*s %-*s %-*s %s\n",
		nameW, "CAMPAIGN", statusW, "STATUS", dmW, "DM", "DATE")
	sb.WriteString(strings.Repeat("-", nameW+statusW+dmW+26) + "\n")
	for _, c := range filtered {
		name := c.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "."
		}
		status := messages.BuildFlags(c)
		if len(status) > statusW {
			status = status[:statusW-1] + "."
		}
		dmName := resolveName(c.DungeonMaster)
		if len(dmName) > dmW {
			dmName = dmName[:dmW-1] + "."
		}
		var date string
		switch {
		case !c.DeletedAt.IsZero():
			date = "deleted " + c.DeletedAt.Format("2006-01-02")
		case c.IsArchived && !c.ArchivedAt.IsZero():
			date = "archived " + c.ArchivedAt.Format("2006-01-02")
		case !c.Schedule.NextSession.IsZero():
			date = "next " + c.Schedule.NextSession.Format("2006-01-02")
		default:
			date = "(no date)"
		}
		fmt.Fprintf(&sb, "%-*s %-*s %-*s %s\n",
			nameW, name, statusW, status, dmW, dmName, date)
	}
	block := "```\n" + sb.String() + "```"
	if len(block) > 1900 {
		block = block[:1896] + "```"
	}

	var options []discordgo.SelectMenuOption
	for idx, c := range filtered {
		if idx >= 25 {
			break
		}
		dmName := resolveName(c.DungeonMaster)
		options = append(options, discordgo.SelectMenuOption{
			Label:       c.Name,
			Description: fmt.Sprintf("DM: %s - %s", dmName, messages.BuildFlags(c)),
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
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := values[0]

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
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
					router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
					discordgo.Button{
						Label: messages.AdminContactDMLabel,
						Style: discordgo.LinkButton,
						URL:   fmt.Sprintf("discord://-/users/%s", campaign.DungeonMaster),
					},
					discordgo.Button{
						Label:    messages.AdminRepostBillboardLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.AdminRepostBillboardPrefix, campaign.ID, i.GuildID),
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
adminRepostBillboardHandler opens the billboard channel selector for any campaign.

Reuses importBillboardStep3Components, giving three options: pick a forum channel,
link an existing thread by ID, or auto-create.

CustomID: admin_repost_billboard:<campaignID>:<guildID>
*/
type adminRepostBillboardHandler struct {
	db *bun.DB
}

func (h *adminRepostBillboardHandler) CustomIDPrefix() string {
	return messages.AdminRepostBillboardPrefix
}

func (h *adminRepostBillboardHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[1]

	if authOK, err := auth.Authorize(h.db, helpers.GetUserID(i), auth.ScopeMod, ""); err != nil || !authOK {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	prompt := fmt.Sprintf("**Repost billboard for %s**\n\n", campaign.Name) + messages.ImportBillboardPrompt
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    prompt,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: importBillboardStep3Components(campaignID, i.GuildID),
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
