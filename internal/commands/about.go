package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

// aboutCommand returns a string with attributions, copyright, licences, and a little awoo from the bot.
type aboutCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (a *aboutCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AboutCommandName,
		Description: messages.AboutCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (a *aboutCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	textCopyright := fmt.Sprintf("%s\nCopyright %s. Released under the %s license.\n", messages.AboutCommandBotDesc, messages.AboutCommandCopyright, messages.AboutCommandLicense)
	textAttributions := fmt.Sprintf("Check out the repo here: %s\n**%s**", messages.AboutCommandGitHubRepoLink, messages.AboutCommandAttributions)
	textFooter := fmt.Sprintf("%s\n%s", messages.AboutCommandHelp, messages.AboutCommandAwoo)

	text := fmt.Sprintf("%s\n%s\n%s", textCopyright, textAttributions, textFooter)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: text,
		},
	})
}
