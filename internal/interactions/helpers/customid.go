package helpers

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/messages"
)

/*
SplitCustomID splits a colon-delimited Discord CustomID and ensures it has at
least n parts. On failure it sends InvalidButtonDataMessage and returns
(nil, false), so the caller can just early-return.

Use it for both component CustomIDs (i.MessageComponentData().CustomID) and
modal CustomIDs (i.ModalSubmitData().CustomID): the caller passes the string.
*/
func SplitCustomID(s *discordgo.Session, i *discordgo.InteractionCreate, customID string, n int) ([]string, bool) {
	parts := strings.SplitN(customID, ":", n)
	if len(parts) < n {
		Respond(s, i, messages.InvalidButtonDataMessage)
		return nil, false
	}
	return parts, true
}

/*
GetUserID returns the invoking user's Discord ID, handling both guild (Member)
and DM (User) contexts. Empty string if neither is populated.
*/
func GetUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}
