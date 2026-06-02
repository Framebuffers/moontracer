package interactions

/*
	Admin Settings handlers for the /admin hub -two pages.

	Page 1 (General): campaign category + announcement channel.
	Page 2 (Billboard): per-format forum channels.

	Discord allows max 5 ActionRows per message. Each SelectMenu occupies its
	own row, so 5 selects would leave no room for navigation. Splitting into two
	pages keeps each page well under the limit.

	Flow:
		1. Staff clicks "Settings" in the /admin hub.
		2. Auth: ScopeAdmin.
		3. Page 1 loads: category select + campaign channel select + nav row.
		4. Staff picks "Billboard channels ->" to open page 2.
		5. Page 2: three forum-channel selects (one per format) + back row.
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

func channelRef(id string) string {
	if id == "" {
		return messages.AdminBillboardNotSet
	}
	return fmt.Sprintf(messages.AdminBillboardCurrentFmt, "<#"+id+">")
}

/*
	1. General Settings
*/

func renderAdminGeneralSettings(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB) {
	settings, err := models.GetOrCreateGuildSettings(db)
	if err != nil {
		log.Printf("admin_settings: load guild settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	content := messages.AdminSettingsGeneralHeader +
		fmt.Sprintf("**%s:** %s\n**%s:** %s",
			messages.AdminCampaignsCategoryLabel, channelRef(settings.CampaignsCategoryID),
			messages.AdminCampaignChannelLabel, channelRef(settings.CampaignChannelID),
		)

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

	components := []discordgo.MessageComponent{
		chanSel(messages.AdminCampaignsCategorySetPrefix, messages.AdminCampaignsCategoryPlaceholder,
			discordgo.ChannelTypeGuildCategory),
		chanSel(messages.AdminCampaignChannelSetPrefix, messages.AdminCampaignChannelPlaceholder,
			discordgo.ChannelTypeGuildText),
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.BackButton(messages.BackLabel, router.ViewAdmin),
			router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
			router.NavButton(messages.AdminBillboardChannelsLabel, discordgo.PrimaryButton, router.ViewAdminBillboard),
		}},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

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
	renderAdminGeneralSettings(s, i, h.db)
}

/*
adminBillboardSetCategoryHandler persists the admin's category selection.

CustomID: admin_billboard_set_category
*/
type adminBillboardSetCategoryHandler struct {
	db *bun.DB
}

func (h *adminBillboardSetCategoryHandler) CustomIDPrefix() string {
	return messages.AdminBillboardSetCategoryPrefix
}

func (h *adminBillboardSetCategoryHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		log.Printf("admin_billboard_set_category: load settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	settings.BillboardCategoryID = values[0]
	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_billboard_set_category: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	renderAdminGeneralSettings(s, i, h.db)
}

/*
adminCampaignsCategoryHandler persists the admin's category choice for new campaign channels.

CustomID: admin_campaigns_category_set
*/
type adminCampaignsCategoryHandler struct {
	db *bun.DB
}

func (h *adminCampaignsCategoryHandler) CustomIDPrefix() string {
	return messages.AdminCampaignsCategorySetPrefix
}

func (h *adminCampaignsCategoryHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		log.Printf("admin_campaigns_category_set: load settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	settings.CampaignsCategoryID = values[0]
	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_campaigns_category_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	renderAdminGeneralSettings(s, i, h.db)
}

/*
adminCampaignChannelSetHandler persists the admin's campaign announcement channel.

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
	channelID := values[0]
	settings.CampaignChannelID = channelID
	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_campaign_channel_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// Lock the feed: deny @everyone SendMessages, allow the bot to post.
	if err := guard.ChannelPermissionSet(s, channelID, i.GuildID,
		discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionSendMessages); err != nil {
		log.Printf("admin_campaign_channel_set: lock channel %s: %v", channelID, err)
	}
	if s.State != nil && s.State.User != nil {
		if err := guard.ChannelPermissionSet(s, channelID, s.State.User.ID,
			discordgo.PermissionOverwriteTypeMember,
			discordgo.PermissionSendMessages|discordgo.PermissionViewChannel, 0); err != nil {
			log.Printf("admin_campaign_channel_set: allow bot in channel %s: %v", channelID, err)
		}
	}

	renderAdminGeneralSettings(s, i, h.db)
}

/*
	2. Billboard forum channels
*/

func renderAdminBillboardSettings(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB) {
	settings, err := models.GetOrCreateGuildSettings(db)
	if err != nil {
		log.Printf("admin_billboard: load guild settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	content := messages.AdminBillboardHeader +
		fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s",
			messages.AdminBillboardCampaignLabel, channelRef(settings.BillboardChannelCampaign),
			messages.AdminBillboardOneshotLabel, channelRef(settings.BillboardChannelOneshot),
			messages.AdminBillboardWestmarchLabel, channelRef(settings.BillboardChannelWestmarch),
		)

	chanSel := func(customID, placeholder string) discordgo.MessageComponent {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:     discordgo.ChannelSelectMenu,
				CustomID:     customID,
				Placeholder:  placeholder,
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildForum},
			},
		}}
	}

	components := []discordgo.MessageComponent{
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatCampaign),
			messages.AdminBillboardCampaignPlaceholder),
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatOneshot),
			messages.AdminBillboardOneshotPlaceholder),
		chanSel(fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, messages.AdminBillboardFormatWestmarch),
			messages.AdminBillboardWestmarchPlaceholder),
		helpers.BackRow(router.ViewAdminSettings),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
adminBillboardSetHandler persists the admin's forum channel selection for one billboard format.

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

	switch format {
	case messages.AdminBillboardFormatCampaign:
		settings.BillboardChannelCampaign = channelID
	case messages.AdminBillboardFormatOneshot:
		settings.BillboardChannelOneshot = channelID
	case messages.AdminBillboardFormatWestmarch:
		settings.BillboardChannelWestmarch = channelID
	default:
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_billboard_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	renderAdminBillboardSettings(s, i, h.db)
}
