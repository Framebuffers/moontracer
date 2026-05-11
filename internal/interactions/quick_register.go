package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

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
