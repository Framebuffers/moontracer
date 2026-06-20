package interactions

/*
	Next Sessions handler for the /me hub.

	Flow:
		1. User clicks "Next Sessions" button on the /me hub.
		2. Load all upcoming sessions the player is a member of, across all approved campaigns.
		3. Sort by scheduled_at ascending (done by the query).
		4. Render a list with campaign name, date/time, and time remaining.
		5. Back button returns to /me hub.
*/

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

type nextSessionsHandler struct {
	db *bun.DB
}

func (h *nextSessionsHandler) CustomIDPrefix() string {
	return messages.NextSessionsPrefix
}

func (h *nextSessionsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	sessions, err := models.GetAllUpcomingSessionsForPlayer(h.db, userID)
	if err != nil {
		log.Printf("next_sessions: failed to load sessions for %s: %v", userID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	backRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.BackButton(messages.BackLabel, router.ViewMe),
	}}

	if len(sessions) == 0 {
		helpers.RespondUpdate(s, i, messages.NextSessionsNone, nil, []discordgo.MessageComponent{backRow})
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("next_sessions: load settings for %s: %v", userID, err)
	}
	loc := time.UTC
	if settings != nil {
		loc = settings.Location()
	}

	var lines []string
	for _, sess := range sessions {
		campaignName := ""
		if sess.Campaign != nil {
			campaignName = sess.Campaign.Name
		}
		formatted := helpers.FormatInLocation(sess.ScheduledAt, messages.SessionListFormat, loc) + " " + helpers.TZLabel(loc)
		lines = append(lines, fmt.Sprintf("• **%s**- %s · %s", campaignName, formatted, helpers.TimeRemaining(sess.ScheduledAt)))
	}
	content := messages.NextSessionsHeader + "\n" + strings.Join(lines, "\n")

	helpers.RespondUpdate(s, i, content, nil, []discordgo.MessageComponent{backRow})
}
