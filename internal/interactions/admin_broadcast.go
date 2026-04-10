package interactions

/*
	Admin Broadcast handler for the /admin hub.

	Flow:
		1. Staff clicks "Broadcast" on the /admin hub.
		2. Auth: ScopeMod.
		3. Open a modal with a text field for the broadcast message.
		4. On submit, resolve all registered players and send DMs via dispatcher.
		5. Confirm with count of messages sent.

	Prerequisites:
		- dispatch.Dispatcher for sending DMs.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	"moontracer/internal/messages"
)

type adminBroadcastHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *adminBroadcastHandler) CustomIDPrefix() string {
	return messages.AdminBroadcastPrefix
}

func (h *adminBroadcastHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement — open the broadcast modal
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeMod, "")
	// 2. respond with InteractionResponseModal
	//    CustomID: messages.AdminBroadcastModalID
	//    Field: message text
	respondInteraction(s, i, "Broadcast is not yet implemented.")
}

type adminBroadcastModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *adminBroadcastModal) CustomIDPrefix() string {
	return messages.AdminBroadcastModalID
}

func (h *adminBroadcastModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeMod, "")
	// 2. extract message text from modal
	// 3. db.GetAll[models.Player](h.db) — all registered players
	// 4. dispatcher.Push for each player
	// 5. respond with success + count
	respondInteraction(s, i, messages.AdminBroadcastFailed)
}
