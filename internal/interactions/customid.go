package interactions

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

/*
splitCustomID splits a colon-delimited Discord CustomID and ensures it has at
least n parts. On failure it sends InvalidButtonDataMessage and returns
(nil, false), so the caller can just early-return.

Use it for both component CustomIDs (i.MessageComponentData().CustomID) and
modal CustomIDs (i.ModalSubmitData().CustomID) — the caller passes the string.
*/
func splitCustomID(s *discordgo.Session, i *discordgo.InteractionCreate, customID string, n int) ([]string, bool) {
	parts := strings.SplitN(customID, ":", n)
	if len(parts) < n {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return nil, false
	}
	return parts, true
}
