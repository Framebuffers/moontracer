package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	msg "moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/help`.
		2. Load all registered commands via `All()`.
		3. Build a formatted list of command names and descriptions.
		4. Respond with the list.
*/

// helpCommand returns a list with all available commands.
type helpCommand struct {
	db bun.DB
	d  *dispatch.Dispatcher
}

// Data is the command metadata that Discord shows to users.
func (h *helpCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        msg.HelpCommandName,
		Description: msg.HelpCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (c *helpCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands := All(&c.db, c.d)

	var helpText strings.Builder
	helpText.WriteString("**Available Commands: **\n")
	for _, cmd := range commands {
		data := cmd.Data()
		fmt.Fprintf(&helpText, "**/%s** - %s\n", data.Name, data.Description)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpText.String(),
		},
	})
}
