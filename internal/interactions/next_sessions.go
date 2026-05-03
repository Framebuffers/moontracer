package interactions

/*
	Next Sessions handler for the /me hub.

	Flow:
		1. User clicks "Next Sessions" button on the /me hub.
		2. Load all CampaignPlayer rows for the user.
		3. Filter to active memberships in approved campaigns whose NextSession
		   is set and in the future.
		4. Sort by NextSession ascending.
		5. Render a list with campaign name, day, and time UTC.
		6. Back button returns to /me hub.
*/

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type nextSessionsHandler struct {
	db *bun.DB
}

func (h *nextSessionsHandler) CustomIDPrefix() string {
	return messages.NextSessionsPrefix
}

func (h *nextSessionsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	entries, err := models.GetPlayerCampaigns(h.db, userID)
	if err != nil {
		log.Printf("next_sessions: failed to load campaigns for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	now := time.Now().UTC()

	type upcoming struct {
		Name string
		When time.Time
	}
	var list []upcoming
	for _, e := range entries {
		if e.Status != models.StatusActive {
			continue
		}
		if e.Campaign == nil || !e.Campaign.IsApproved {
			continue
		}
		if e.Campaign.Schedule.NextSession.IsZero() || !e.Campaign.Schedule.NextSession.After(now) {
			continue
		}
		list = append(list, upcoming{
			Name: e.Campaign.Name,
			When: e.Campaign.Schedule.NextSession.UTC(),
		})
	}

	backRow := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.BackButton(messages.BackLabel, router.ViewMe),
	}}

	if len(list) == 0 {
		helpers.RespondUpdate(s, i, messages.NextSessionsNone, nil, []discordgo.MessageComponent{backRow})
		return
	}

	sort.Slice(list, func(a, b int) bool { return list[a].When.Before(list[b].When) })

	var lines []string
	for _, e := range list {
		lines = append(lines, fmt.Sprintf("• **%s** — %s UTC", e.Name, e.When.Format(messages.SessionListFormat)))
	}
	content := messages.NextSessionsHeader + "\n" + strings.Join(lines, "\n")

	helpers.RespondUpdate(s, i, content, nil, []discordgo.MessageComponent{backRow})
}
