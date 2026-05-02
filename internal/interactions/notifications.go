package interactions

/*
	Notifications handler for the /me hub.

	Flow:
		1. User clicks "Notifications" button on the /me hub.
		2. Load (or create) PlayerSettings for the user.
		3. Render current preferences as toggle buttons:
			- Announcements (green = ON, red = OFF)
			- Session Reminders
			- Invitations
		4. Clicking a toggle flips the stored bool and re-renders.
		5. Back button returns to /me hub.

	CustomID formats:
		notifications                          -- open panel
		notif_toggle:<field>                   -- flip field (announcements|sessions|invitations)
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type notificationsHandler struct {
	db *bun.DB
}

func (h *notificationsHandler) CustomIDPrefix() string {
	return messages.NotificationsPrefix
}

func (h *notificationsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	renderNotificationsPanel(s, i, h.db, userID)
}

// notifToggleHandler flips one notification preference and re-renders the panel.
type notifToggleHandler struct {
	db *bun.DB
}

func (h *notifToggleHandler) CustomIDPrefix() string {
	return messages.NotifTogglePrefix
}

func (h *notifToggleHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	field := parts[1]
	userID := getUserID(i)

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("notif_toggle: load failed for %s: %v", userID, err)
		respondInteraction(s, i, messages.NotifLoadFailed)
		return
	}

	switch field {
	case messages.NotifFieldAnnouncements:
		settings.NotifyAnnouncements = !settings.NotifyAnnouncements
	case messages.NotifFieldSessions:
		settings.NotifySessionRemind = !settings.NotifySessionRemind
	case messages.NotifFieldInvitations:
		settings.NotifyInvitations = !settings.NotifyInvitations
	default:
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}

	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("notif_toggle: update failed for %s: %v", userID, err)
		respondInteraction(s, i, messages.NotifUpdateFailed)
		return
	}

	renderNotificationsPanel(s, i, h.db, userID)
}

// renderNotificationsPanel shows toggle buttons for the user's current settings.
func renderNotificationsPanel(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	settings, err := models.GetOrCreatePlayerSettings(db, userID)
	if err != nil {
		log.Printf("notifications: load failed for %s: %v", userID, err)
		respondInteraction(s, i, messages.NotifLoadFailed)
		return
	}

	row := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		notifToggleButton(messages.NotifFieldAnnouncements, messages.NotifLabelAnnouncements, settings.NotifyAnnouncements),
		notifToggleButton(messages.NotifFieldSessions, messages.NotifLabelSessions, settings.NotifySessionRemind),
		notifToggleButton(messages.NotifFieldInvitations, messages.NotifLabelInvitations, settings.NotifyInvitations),
	}}
	backRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.BackButton(messages.BackLabel, router.ViewMe),
	}}

	respondUpdate(s, i, messages.NotificationsHeader, nil, []discordgo.MessageComponent{row, backRow})
}

func notifToggleButton(field, label string, enabled bool) discordgo.Button {
	style := discordgo.DangerButton
	text := fmt.Sprintf(messages.NotifDisabledSuffix, label)
	if enabled {
		style = discordgo.SuccessButton
		text = fmt.Sprintf(messages.NotifEnabledSuffix, label)
	}
	return discordgo.Button{
		Label:    text,
		Style:    style,
		CustomID: fmt.Sprintf("%s:%s", messages.NotifTogglePrefix, field),
	}
}
