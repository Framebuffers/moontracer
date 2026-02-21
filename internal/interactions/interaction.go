package interactions

import "github.com/bwmarrin/discordgo"

// ComponenHandler defines a single identification string, plus its handler, that all Components must implement.
type ComponentHandler interface {
	CustomIDPrefix() string // convention = prefix:arg1:arg2 (campaign_join:arg-123)
	HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// ModalHandler defines a single identification string, plus its handler, that all Modals must implement. Modals are stored in a registry for then to be declared.
type ModalHandler interface {
	CustomIDPrefix() string
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
}
