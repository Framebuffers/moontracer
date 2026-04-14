package interactions

/*
	navHandler is the single component handler behind every "nav:" CustomID.
	It parses the CustomID, looks up the ViewID in the router registry, and
	invokes the registered RenderFunc.

	Registration of views themselves lives in views.go (RegisterAllViews).
*/

import (
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
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	router.Navigate(s, i, viewID, args)
}
