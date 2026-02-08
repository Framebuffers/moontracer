package commands

import "github.com/bwmarrin/discordgo"

// Command is the interface every slash command must implement.
type Command interface {
	Data() *discordgo.ApplicationCommand
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate)
}
