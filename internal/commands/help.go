package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	msg "github.com/framebuffers/moontracer/internal/messages"
)

/*
	/help lists the bot's slash commands and offers a jump back to /me so a
	lost user always has a single click back to their hub.

	Flow (slash command):
		1. User runs `/help`.
		2. All(&db, d) returns the registered ApplicationCommand list.
		3. helpText is a bulleted list of "/<name> - description" entries.
		4. Respond with source + a "My Hub" button routing to ViewMe.

	Flow (router ViewHelp):
		1. Another view's button navigates to ViewHelp.
		2. views.go adapter calls RenderHelp(s, i, db, d).
		3. RenderHelp rebuilds the same body and responds with
		   InteractionResponseUpdateMessage so it replaces the current view.
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
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: helpResponseData(&c.db, c.d),
	})
}

// RenderHelp re-renders /help as a message update (for router back/forward nav).
func RenderHelp(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, d *dispatch.Dispatcher) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: helpResponseData(db, d),
	})
}

func helpResponseData(db *bun.DB, d *dispatch.Dispatcher) *discordgo.InteractionResponseData {
	commands := All(db, d, "", "")

	var helpText strings.Builder
	helpText.WriteString("**Available Commands: **\n")
	for _, cmd := range commands {
		if h, ok := cmd.(HiddenCommand); ok && h.Hidden() {
			continue
		}
		data := cmd.Data()
		fmt.Fprintf(&helpText, "**/%s** - %s\n", data.Name, data.Description)
	}

	return &discordgo.InteractionResponseData{
		Content: helpText.String(),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				router.NavButton(msg.MyProfileLabel, discordgo.PrimaryButton, router.ViewMe),
			}},
		},
		Flags: discordgo.MessageFlagsEphemeral,
	}
}
