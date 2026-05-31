package interactions

/*
	Admin Settings handler for the /admin hub.

	Flow:
		1. Staff clicks "Settings" on the /admin hub.
		2. Auth: ScopeAdmin.
		3. Load GuildSettings; show three channel-select menus (one per billboard format).
		4. Each select fires adminBillboardSetHandler, which persists the chosen channel ID.
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

type adminSettingsHandler struct {
	db *bun.DB
}

func (h *adminSettingsHandler) CustomIDPrefix() string {
	return messages.AdminSettingsPrefix
}

func (h *adminSettingsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	settings, err := models.GetOrCreateGuildSettings(h.db)
	if err != nil {
		log.Printf("admin_settings: load guild settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	content := messages.AdminBillboardHeader + buildBillboardStatus(settings)
	helpers.RespondWithBack(s, i, discordgo.InteractionResponseUpdateMessage, content,
		buildSettingsComponents(), router.ViewAdmin)
}

// buildBillboardStatus formats current settings (channel IDs or "not set") for display.
// BillboardCategoryID is auto-derived and shown read-only; admins set it by picking any forum channel.
func buildBillboardStatus(s *models.GuildSettings) string {
	ch := func(id string) string {
		if id == "" {
			return messages.AdminBillboardNotSet
		}
		return fmt.Sprintf(messages.AdminBillboardCurrentFmt, "<#"+id+">")
	}
	return fmt.Sprintf(
		"**%s:** %s\n**%s:** %s\n**%s:** %s\n**%s:** %s\n**%s:** %s",
		messages.AdminBillboardCategoryLabel, ch(s.BillboardCategoryID),
		messages.AdminCampaignChannelLabel, ch(s.CampaignChannelID),
		messages.AdminBillboardCampaignLabel, ch(s.BillboardChannelCampaign),
		messages.AdminBillboardOneshotLabel, ch(s.BillboardChannelOneshot),
		messages.AdminBillboardWestmarchLabel, ch(s.BillboardChannelWestmarch),
	) + "\n\n-# Category is set automatically when you pick a forum channel above."
}

/*
buildSettingsComponents renders the four channel selects for the settings panel.

Category is auto-derived from the parent of whichever forum channel the admin sets first.
*/
func buildSettingsComponents() []discordgo.MessageComponent {
	chanSel := func(customID, placeholder string, types ...discordgo.ChannelType) discordgo.MessageComponent {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:     discordgo.ChannelSelectMenu,
				CustomID:     customID,
				Placeholder:  placeholder,
				ChannelTypes: types,
			},
		}}
	}
	return []discordgo.MessageComponent{
		chanSel(messages.AdminCampaignChannelSetPrefix, messages.AdminCampaignChannelPlaceholder,
			discordgo.ChannelTypeGuildText),
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatCampaign),
			messages.AdminBillboardCampaignPlaceholder, discordgo.ChannelTypeGuildForum),
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatOneshot),
			messages.AdminBillboardOneshotPlaceholder, discordgo.ChannelTypeGuildForum),
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatWestmarch),
			messages.AdminBillboardWestmarchPlaceholder, discordgo.ChannelTypeGuildForum),
	}
}

/*
adminBillboardSetHandler persists the admin's channel selection for one billboard format.

CustomID: admin_billboard_set:<format>{campaign, oneshot, westmarch}
*/
type adminBillboardSetHandler struct {
	db *bun.DB
}

func (h *adminBillboardSetHandler) CustomIDPrefix() string {
	return messages.AdminBillboardSetPrefix
}

func (h *adminBillboardSetHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	format := parts[1]

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	channelID := values[0]

	settings, err := models.GetOrCreateGuildSettings(h.db)
	if err != nil {
		log.Printf("admin_billboard_set: load settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	var formatLabel string
	switch format {
	case messages.AdminBillboardFormatCampaign:
		settings.BillboardChannelCampaign = channelID
		formatLabel = messages.AdminBillboardCampaignLabel
	case messages.AdminBillboardFormatOneshot:
		settings.BillboardChannelOneshot = channelID
		formatLabel = messages.AdminBillboardOneshotLabel
	case messages.AdminBillboardFormatWestmarch:
		settings.BillboardChannelWestmarch = channelID
		formatLabel = messages.AdminBillboardWestmarchLabel
	default:
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if ch, err := s.Channel(channelID); err == nil && ch.ParentID != "" {
		settings.BillboardCategoryID = ch.ParentID
	}

	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_billboard_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	content := fmt.Sprintf(messages.AdminBillboardSavedFmt, formatLabel) +
		"\n\n" + messages.AdminBillboardHeader + buildBillboardStatus(settings)
	helpers.RespondWithBack(s, i, discordgo.InteractionResponseUpdateMessage, content,
		buildSettingsComponents(), router.ViewAdmin)
}

/*
adminCampaignChannelSetHandler persists the admin's campaign channel selection.

CustomID: admin_campaign_channel_set
*/
type adminCampaignChannelSetHandler struct {
	db *bun.DB
}

func (h *adminCampaignChannelSetHandler) CustomIDPrefix() string {
	return messages.AdminCampaignChannelSetPrefix
}

func (h *adminCampaignChannelSetHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	if ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, ""); err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	settings, err := models.GetOrCreateGuildSettings(h.db)
	if err != nil {
		log.Printf("admin_campaign_channel_set: load settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	settings.CampaignChannelID = values[0]
	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_campaign_channel_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	content := fmt.Sprintf(messages.AdminCampaignChannelSavedFmt, "<#"+values[0]+">") +
		"\n\n" + messages.AdminBillboardHeader + buildBillboardStatus(settings)
	helpers.RespondWithBack(s, i, discordgo.InteractionResponseUpdateMessage, content,
		buildSettingsComponents(), router.ViewAdmin)
}
