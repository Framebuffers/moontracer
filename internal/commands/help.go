package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	msg "moontracer/internal/messages"
)

// helpCommand returns a list with all available commands.
type helpCommand struct {
	db bun.DB
}

func (h *helpCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        msg.HelpCommandName,
		Description: msg.HelpCommandDesc,
	}
}

func (c *helpCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands := All(&c.db)

	helpText := "**Available Commands: **\n"
	for _, cmd := range commands {
		data := cmd.Data()
		helpText += fmt.Sprintf("**/%s** - %s\n", data.Name, data.Description)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpText,
		},
	})
}
