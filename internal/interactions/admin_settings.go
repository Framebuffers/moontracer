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
		buildBillboardComponents(), router.ViewAdmin)
}

// buildBillboardStatus formats the current channel IDs (or "not set") for display.
func buildBillboardStatus(s *models.GuildSettings) string {
	label := func(chID string) string {
		if chID == "" {
			return messages.AdminBillboardNotSet
		}
		return fmt.Sprintf(messages.AdminBillboardCurrentFmt, "<#"+chID+">")
	}
	return fmt.Sprintf("**%s:** %s\n**%s:** %s\n**%s:** %s",
		messages.AdminBillboardCampaignLabel, label(s.BillboardChannelCampaign),
		messages.AdminBillboardOneshotLabel, label(s.BillboardChannelOneshot),
		messages.AdminBillboardWestmarchLabel, label(s.BillboardChannelWestmarch),
	)
}

// buildBillboardComponents renders three channel-select menus, one per format.
func buildBillboardComponents() []discordgo.MessageComponent {
	sel := func(format, placeholder string) discordgo.MessageComponent {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.ChannelSelectMenu,
				CustomID:    fmt.Sprintf("%s:%s", messages.AdminBillboardSetPrefix, format),
				Placeholder: placeholder,
			},
		}}
	}
	return []discordgo.MessageComponent{
		sel(messages.AdminBillboardFormatCampaign, messages.AdminBillboardCampaignPlaceholder),
		sel(messages.AdminBillboardFormatOneshot, messages.AdminBillboardOneshotPlaceholder),
		sel(messages.AdminBillboardFormatWestmarch, messages.AdminBillboardWestmarchPlaceholder),
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

	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("admin_billboard_set: save settings: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	content := fmt.Sprintf(messages.AdminBillboardSavedFmt, formatLabel) +
		"\n\n" + messages.AdminBillboardHeader + buildBillboardStatus(settings)
	helpers.RespondWithBack(s, i, discordgo.InteractionResponseUpdateMessage, content,
		buildBillboardComponents(), router.ViewAdmin)
}
