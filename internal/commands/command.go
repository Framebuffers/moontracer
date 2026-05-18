package commands

import "github.com/bwmarrin/discordgo"

// Command is the interface every slash command must implement.
type Command interface {
	Data() *discordgo.ApplicationCommand
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// AutocompleteCommand is an optional extension for commands that support Discord autocomplete.
type AutocompleteCommand interface {
	Command
	Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate)
}

/*
HiddenCommand is an optional marker for commands that should be omitted from /help output.
Use for staff-only or low-signal commands that clutter the help list for regular users.
*/
type HiddenCommand interface {
	Command
	Hidden() bool
}
