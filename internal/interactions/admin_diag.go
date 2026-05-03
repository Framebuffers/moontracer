package interactions

/*
	Admin Diagnostics handler for the /admin hub.

	Flow:
		1. Staff clicks "Diagnostics" on the /admin hub.
		2. Auth: ScopeMod.
		3. Render runtime / session / config diagnostics as an ephemeral sub-view.
		4. Back button returns to /admin hub via BackAdminID.
*/

import (
	"moontracer/internal/interactions/helpers"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/commands"
	"moontracer/internal/guard"
	"moontracer/internal/messages"
)

type adminDiagHandler struct {
	db *bun.DB
}

func (h *adminDiagHandler) CustomIDPrefix() string {
	return messages.AdminDiagPrefix
}

func (h *adminDiagHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	/*
		Double-gate:
			The button is hidden in adminHubData when DevMode is off,
			but a stale ephemeral could still carry the CustomID.

			Reject here so production never exposes the raw database view regardless.
	*/
	if !guard.DevMode {
		helpers.Respond(s, i, messages.DebugSurfaceDisabled)
		return
	}

	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("admin_diag: auth check failed: %v", err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		helpers.Respond(s, i, messages.AdminNotStaff)
		return
	}

	commands.RenderAdminDiag(s, i, h.db, i.GuildID)
}
