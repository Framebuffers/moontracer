package interactions

/*
	Admin Settings handler for the /admin hub.

	Status: stubbed for 1.0. The full settings UI requires a new GuildSettings
	model and migration; that work is deferred to v1.1. For 1.0, the button
	renders a placeholder panel with a back button so the admin hub stays
	navigable without dead-ending.

	Future flow (v1.1):
		1. Staff clicks "Settings" on the /admin hub.
		2. Auth: ScopeAdmin (admin only, not mod).
		3. Show per-guild settings as toggle buttons or a form.
		4. Persist via GuildSettings table (TBD model).
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
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

	ok, err := auth.Authorize(h.db, userID, auth.ScopeAdmin, "")
	if err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	helpers.RespondWithBack(
		s, i,
		discordgo.InteractionResponseUpdateMessage,
		"**Settings**\n\n_The settings UI is coming in v1.1. Per-guild configuration (default reminder times, notification channels, auto-approval policy) will live here._",
		nil,
		router.ViewAdmin,
	)
}
