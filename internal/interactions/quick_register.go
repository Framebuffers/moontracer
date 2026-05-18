package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		Triggered when an unregistered user clicks a "Register" CTA (call-to-action) button that
		any gated surface can attach (campaign join, /me hub entry, etc.).
		1. Button click (quick_register): quickRegisterHandler
			a. Probes ScopePlayer auth to decide if a Player row already exists.
			b. If not registered: inserts a fresh Player{ID:userID}. Failure ->
			   RegistrationFailureMessage terminal reply.
			c. Either way (already-registered users get the same UX), navigates
			   the viewer to ViewMe via router.Navigate.

	Notes:
		- Idempotent on purpose: a stale button still works for an already-
		  registered user. This just sends them to /me.
		- No /register slash command call here. This is the inline shortcut.
*/

/*
quickRegisterHandler registers the user on the spot and navigates them to the /me hub.

Shown as a button whenever an unregistered user hits a gated surface.

CustomID: quick_register
*/
type quickRegisterHandler struct {
	db *bun.DB
}

func (h *quickRegisterHandler) CustomIDPrefix() string { return messages.QuickRegisterPrefix }

func (h *quickRegisterHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	registered, err := auth.Authorize(h.db, userID, auth.ScopePlayer, "")
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if !registered {
		player := &models.Player{ID: userID}
		if err := db.Insert(h.db, player); err != nil {
			log.Printf("quick_register: insert failed for %s: %v", userID, err)
			helpers.RespondUpdateTerminal(s, i, messages.RegistrationFailureMessage)
			return
		}
	}

	router.Navigate(s, i, router.ViewMe, nil)
}
