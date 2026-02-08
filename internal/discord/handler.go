package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/commands"
)

// NewHandler returns a discordgo event handler that dispatches interaction
// creates to the matching Command.Execute.
func NewHandler(cmds []commands.Command) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lookup := make(map[string]commands.Command, len(cmds))
	for _, cmd := range cmds {
		lookup[cmd.Data().Name] = cmd
	}

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		name := i.ApplicationCommandData().Name
		cmd, ok := lookup[name]
		if !ok {
			log.Printf("unknown command: /%s", name)
			return
		}
		cmd.Execute(s, i)
	}
}
