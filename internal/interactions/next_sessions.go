package interactions

/*
	Next Sessions handler for the /me hub.

	Flow:
		1. User clicks "Next Sessions" button on the /me hub.
		2. Load all campaigns where the user is an active member.
		3. Filter to campaigns with a NextSession set and in the future.
		4. Sort by NextSession ascending.
		5. Render a list with campaign name, date, and time.
		6. Back button returns to /me hub.
*/

import (
	"moontracer/internal/interactions/helpers"
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type nextSessionsHandler struct {
	db *bun.DB
}

func (h *nextSessionsHandler) CustomIDPrefix() string {
	return messages.NextSessionsPrefix
}

func (h *nextSessionsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. helpers.GetUserID(i)
	// 2. models.GetPlayerCampaigns(h.db, userID)
	// 3. filter: Campaign.Schedule.NextSession > now, Status == active
	// 4. sort by NextSession
	// 5. render list + back button (messages.BackMeID)
	helpers.Respond(s, i, messages.NextSessionsNone)
}
