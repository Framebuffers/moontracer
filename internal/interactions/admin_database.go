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
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
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
	userID := i.Member.User.ID
	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		return
	}

	campaigns, err := db.GetAll[models.Campaign](h.db)
	if err != nil {
		log.Printf("campaign_database: failed to load campaigns: %v", err)
		return
	}

	if len(campaigns) == 0 {
		return
	}

	var lines []string
	for _, camp := range campaigns {
		flags := messages.BuildFlags(camp)
		lines = append(lines, truncate(fmt.Sprintf("**%s** (`%s`) — DM: <@%s> [%s]",
			camp.Name, camp.Tag, camp.DungeonMaster, flags), 1900))
	}
	_ = lines

	respondInteraction(s, i, "Database view is not yet implemented.")
}
