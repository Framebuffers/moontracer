package interactions

import "github.com/bwmarrin/discordgo"

type ComponentHandler interface {
	CustomIDPrefix() string // convention = prefix:arg1:arg2 (campaign_join:arg-123)
	HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type ModalHandler interface {
	CustomIDPrefix() string
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
}
