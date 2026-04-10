package interactions

/*
	Notifications handler for the /me hub.

	Flow:
		1. User clicks "Notifications" button on the /me hub.
		2. Show current notification preferences (requires a PlayerSettings model — does not exist yet).
		3. Allow toggling: session reminders, campaign announcements, join approvals.
		4. Back button returns to /me hub.

	Prerequisites:
		- New model: PlayerSettings (or NotificationPrefs) with per-player toggles.
		- New DB table + migration.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type notificationsHandler struct {
	db *bun.DB
}

func (h *notificationsHandler) CustomIDPrefix() string {
	return messages.NotificationsPrefix
}

func (h *notificationsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. getUserID(i)
	// 2. load PlayerSettings from DB (create default if not exists)
	// 3. render toggles as buttons (enabled/disabled style)
	// 4. back button (messages.BackMeID)
	respondInteraction(s, i, messages.NotificationsNone)
}
