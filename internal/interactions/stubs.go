package interactions

import (
	"github.com/bwmarrin/discordgo"
)

// stubHandler responds to all stub_* component interactions with a placeholder message.
type stubHandler struct{}

func (h *stubHandler) CustomIDPrefix() string {
	return "stub"
}

func (h *stubHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	respondInteraction(s, i, "This feature is coming soon!")
}
