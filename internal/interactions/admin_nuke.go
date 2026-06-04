package interactions

/*
Admin Nuke handler

Hard-deletes a campaign from the DB and Discord.

Flow:
 1. Staff clicks the "nuke campaign" button in the /admin hub.
 2. Auth: ScopeAdmin (stricter than ScopeMod, it's a `moontracer-admin`-only command).
 3. Campaign selector shows ALL campaigns (same query as adminCampaignsHandler).
 4. Staff picks a campaign, adminNukeSelectHandler shows a confirmation embed.
 5. Staff clicks "CONFIRM NUKE", then adminNukeConfirmHandler executes:
    a. Write audit entry (campaign_nuke).
    b. Safety check: fetch Discord channel, assert ID matches campaign.ChannelID.
    c. Delete Discord role (if set).
    d. Delete billboard forum thread (if set) (guarded by DeleteBillboard).
    e. Delete Discord campaign channel (only if IDs match)
    f. PurgeCampaignData: delete sessions, responses, players.
    g. Hard-delete the campaign row (ForceDelete, bypasses soft-delete).
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auditlog"
	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// adminNukeHandler opens the campaign selector for the nuke flow.
type adminNukeHandler struct {
	db *bun.DB
}

func (h *adminNukeHandler) CustomIDPrefix() string { return messages.AdminNukePrefix }

func (h *adminNukeHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	var campaigns []models.Campaign
	err := h.db.NewSelect().
		Model(&campaigns).
		WhereAllWithDeleted().
		OrderExpr("deleted_at IS NOT NULL ASC, is_archived ASC, name ASC").
		Scan(context.Background())
	if err != nil || len(campaigns) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.AdminCampaignsNone)
		return
	}

	var options []discordgo.SelectMenuOption
	for idx, c := range campaigns {
		if idx >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       c.Name,
			Description: fmt.Sprintf("Tag: %s — DM: %s", c.Tag, c.DungeonMaster),
			Value:       c.ID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ **Nuke Campaign** — Select a campaign to permanently delete:",
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    messages.AdminNukeSelectPrefix,
						Placeholder: messages.AdminNukeSelectPlaceholder,
						Options:     options,
					},
				}},
				helpers.BackRow(router.ViewAdmin),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// adminNukeSelectHandler shows a confirmation embed for the selected campaign.
type adminNukeSelectHandler struct {
	db *bun.DB
}

func (h *adminNukeSelectHandler) CustomIDPrefix() string { return messages.AdminNukeSelectPrefix }

func (h *adminNukeSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := values[0]

	// nuke even soft-deleted campaigns
	campaign := &models.Campaign{}
	if err := h.db.NewSelect().Model(campaign).WhereAllWithDeleted().Where("id = ?", campaignID).Scan(context.Background()); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	confirmText := fmt.Sprintf(messages.AdminNukeConfirmFmt, campaign.Name, campaign.Tag)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: confirmText,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "💣 CONFIRM NUKE",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.AdminNukeConfirmPrefix, campaignID),
					},
					discordgo.Button{
						Label:    "❌ Cancel",
						Style:    discordgo.SecondaryButton,
						CustomID: messages.AdminNukeCancelPrefix,
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// adminNukeCancelHandler returns to the admin hub.
type adminNukeCancelHandler struct{}

func (h *adminNukeCancelHandler) CustomIDPrefix() string { return messages.AdminNukeCancelPrefix }

func (h *adminNukeCancelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	router.Navigate(s, i, router.ViewAdmin, nil)
}

// adminNukeConfirmHandler executes the nuke after the admin confirms.
type adminNukeConfirmHandler struct {
	db *bun.DB
}

func (h *adminNukeConfirmHandler) CustomIDPrefix() string { return messages.AdminNukeConfirmPrefix }

func (h *adminNukeConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	campaign := &models.Campaign{}
	if err := h.db.NewSelect().Model(campaign).WhereAllWithDeleted().Where("id = ?", campaignID).Scan(context.Background()); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    messages.AdminNukeInProgress,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	// Write audit entry first, so if it all fails, at least there's a record of it.
	auditlog.Post(s, h.db, i.GuildID, campaign.DungeonMaster, userID, models.AuditCampaignNuke,
		fmt.Sprintf("admin nuke of campaign %s (%s)", campaign.Name, campaign.Tag))

	// verify Discord channel ID matches DB before deleting.
	if messages.IsSnowflake(campaign.ChannelID) {
		ch, err := s.Channel(campaign.ChannelID)
		if err != nil {
			log.Printf("admin_nuke: fetch channel %s for campaign %s: %v", campaign.ChannelID, campaignID, err)
			helpers.EditTerminal(s, i, messages.AdminNukeChannelMismatch)
			return
		}
		if ch.ID != campaign.ChannelID {
			log.Printf("admin_nuke: channel ID mismatch: DB=%s Discord=%s", campaign.ChannelID, ch.ID)
			helpers.EditTerminal(s, i, messages.AdminNukeChannelMismatch)
			return
		}
	}

	// delete role
	if messages.IsSnowflake(campaign.RoleID) {
		if err := guard.GuildRoleDelete(s, i.GuildID, campaign.RoleID); err != nil {
			log.Printf("admin_nuke: delete role %s for campaign %s: %v", campaign.RoleID, campaignID, err)
		}
	}

	// delete billboard forum thread
	DeleteBillboard(s, campaign)

	// delete the campaign's Discord channel.
	if messages.IsSnowflake(campaign.ChannelID) {
		if _, err := guard.ChannelDelete(s, campaign.ChannelID); err != nil {
			log.Printf("admin_nuke: delete channel %s for campaign %s: %v", campaign.ChannelID, campaignID, err)
		}
	}

	// purge relational data (sessions, players, etc.).
	if err := models.PurgeCampaignData(h.db, campaign); err != nil {
		log.Printf("admin_nuke: purge data for campaign %s: %v", campaignID, err)
	}

	// hard-delete the campaign row, bypassing bun's soft-delete filter.
	if _, err := h.db.NewDelete().Model(campaign).WherePK().ForceDelete().Exec(context.Background()); err != nil {
		log.Printf("admin_nuke: hard-delete campaign %s: %v", campaignID, err)
		helpers.EditTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	log.Printf("admin_nuke: %s nuked campaign %s (%s)", userID, campaign.Name, campaignID)
	helpers.EditTerminal(s, i, fmt.Sprintf(messages.AdminNukeSuccess, campaign.Name))
}
