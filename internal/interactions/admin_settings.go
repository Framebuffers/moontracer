package interactions

/*
	Admin Settings handler for the /admin hub.

	Flow:
		1. Staff clicks "Settings" on the /admin hub.
		2. Auth: ScopeAdmin (admin only, not mod).
		3. Show bot/guild-level settings as toggle buttons or a form.
		4. Back button returns to /admin hub.

	Prerequisites:
		- New model: GuildSettings with per-guild configuration.
		- New DB table + migration.
		- Possible settings: default session reminder time, auto-approve campaigns,
		  notification channel, etc.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type adminSettingsHandler struct {
	db *bun.DB
}

func (h *adminSettingsHandler) CustomIDPrefix() string {
	return messages.AdminSettingsPrefix
}

func (h *adminSettingsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeAdmin, "")
	// 2. load GuildSettings from DB (create default if not exists)
	// 3. render settings as buttons/toggles
	// 4. back button (messages.BackAdminID)
	respondInteraction(s, i, "Settings are not yet implemented.")
}
