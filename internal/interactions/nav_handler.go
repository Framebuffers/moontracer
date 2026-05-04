package interactions

/*
	navHandler is the single component handler behind every "nav:" CustomID.

	Flow:
		1. User clicks any button whose CustomID starts with "nav:".
		2. discordgo's prefix dispatcher matches on CustomIDPrefix() == "nav"
		   and invokes HandleComponents on this handler.
		3. HandleComponents calls router.ParseCustomID to split the CustomID
		   into (ViewID, args). Malformed IDs produce an ephemeral error so
		   the user isn't left on a spinning interaction.
		4. router.Navigate looks up the ViewID in the registry (populated at
		   startup by RegisterAllViews in views.go) and invokes the
		   registered RenderFunc, which renders the target view in place.

	This handler is the only piece of glue between Discord's component
	dispatch and the router. Registration of views themselves lives in
	views.go.
*/

import (
	"moontracer/internal/interactions/helpers"
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/router"
	"moontracer/internal/messages"
)

type navHandler struct {
	db *bun.DB
}

func (h *navHandler) CustomIDPrefix() string {
	return router.NavPrefix
}

func (h *navHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	viewID, args, ok := router.ParseCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	router.Navigate(s, i, viewID, args)
}
