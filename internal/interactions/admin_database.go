package interactions

/*
	Admin Database handler for the /admin hub.

	Flow:
		1. Staff clicks "Database" on the /admin hub.
		2. Auth: ScopeMod.
		3. Load all campaigns from DB (same logic as campaignDatabaseCommand).
		4. Build a formatted list with name, tag, DM, status flags.
		5. Respond ephemerally.
		6. Back button returns to /admin hub.

	Note: this replicates the existing /campaigndatabase slash command as a button handler.
	Once fully wired, the slash command can be removed.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/guard"
	"moontracer/internal/messages"
)

type adminDatabaseHandler struct {
	db *bun.DB
}

func (h *adminDatabaseHandler) CustomIDPrefix() string {
	return messages.AdminDatabasePrefix
}

func (h *adminDatabaseHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	/*
		Double-gate:
			The button is hidden in adminHubData when DevMode is off,
			but a stale ephemeral could still carry the CustomID.

			Reject here so production never exposes the raw database view regardless.
	*/
	if !guard.DevMode {
		respondInteraction(s, i, messages.DebugSurfaceDisabled)
		return
	}

	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeMod, "")
	// 2. db.GetAll[models.Campaign](h.db)
	// 3. build formatted list (reuse buildFlags pattern from campaign_database.go)
	// 4. truncate at 1900 chars
	// 5. respond with list + back button (messages.BackAdminID)
	respondInteraction(s, i, "Database view is not yet implemented.")
}
