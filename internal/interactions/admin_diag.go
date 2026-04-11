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
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/commands"
	"moontracer/internal/messages"
)

type adminDiagHandler struct {
	db *bun.DB
}

func (h *adminDiagHandler) CustomIDPrefix() string {
	return messages.AdminDiagPrefix
}

func (h *adminDiagHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("admin_diag: auth check failed: %v", err)
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondInteraction(s, i, messages.AdminNotStaff)
		return
	}

	commands.RenderAdminDiag(s, i)
}
