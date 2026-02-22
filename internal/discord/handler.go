package discord

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/commands"
	"moontracer/internal/interactions"
)

/*
	Flow:
		1. NewHandler receives all registered commands, component handlers, and modal handlers.
		2. Builds three lookup maps: cmdLookup (by command name), compLookup (by CustomIDPrefix), modalLookup (by CustomIDPrefix).
		3. Returns a closure that acts as the main Discord event handler — this closure receives ALL interactions.
		4. For each interaction:
			a. If ApplicationCommand: look up by command name, call Execute().
			b. If MessageComponent (button): extract prefix from CustomID (split on ":"), look up handler, call HandleComponents().
			c. If ModalSubmit: extract prefix from CustomID, look up handler, call HandleModal().
		5. Unknown interactions are logged and ignored.
*/

// NewHandler returns a discordgo event handler that dispatches slash commands,
// component interactions (buttons), and modal submissions.
func NewHandler(cmds []commands.Command, components []interactions.ComponentHandler, modals []interactions.ModalHandler) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdLookup := make(map[string]commands.Command, len(cmds))
	for _, cmd := range cmds {
		cmdLookup[cmd.Data().Name] = cmd
	}

	compLookup := make(map[string]interactions.ComponentHandler, len(components))
	for _, c := range components {
		compLookup[c.CustomIDPrefix()] = c
	}

	modalLookup := make(map[string]interactions.ModalHandler, len(modals))
	for _, m := range modals {
		modalLookup[m.CustomIDPrefix()] = m
	}

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			name := i.ApplicationCommandData().Name
			cmd, ok := cmdLookup[name]
			if !ok {
				log.Printf("unknown command: /%s", name)
				return
			}
			cmd.Execute(s, i)

		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			prefix := customID
			if idx := strings.Index(customID, ":"); idx != -1 {
				prefix = customID[:idx]
			}
			handler, ok := compLookup[prefix]
			if !ok {
				log.Printf("unknown component: %s", customID)
				return
			}
			handler.HandleComponents(s, i)

		case discordgo.InteractionModalSubmit:
			customID := i.ModalSubmitData().CustomID
			prefix := customID
			if idx := strings.Index(customID, ":"); idx != -1 {
				prefix = customID[:idx]
			}
			handler, ok := modalLookup[prefix]
			if !ok {
				log.Printf("unknown modal: %s", customID)
				return
			}
			handler.HandleModal(s, i)
		}
	}
}
