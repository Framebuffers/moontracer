package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/router"
	"moontracer/internal/messages"
)

/*
	The bot's about page (/moontracer).

	Renders attribution, copyright, license, and three extra buttons so new users
	have an obvious next step, instead of a dead-end wall of text:
	"My Hub" (ViewMe), "Help" (ViewHelp), and a link button to the GitHub
	repo (external URL, so Style=Link with no CustomID).

	Flow:
		1. User runs `/moontracer`.
		2. Assemble body (description, copyright/license, repo line, attributions, footer).
		3. Respond with source + three-button action row.
*/

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
	// textCopyright := fmt.Sprintf("%s\nCopyright %s. Released under the %s license.\n", messages.AboutCommandBotDesc, messages.AboutCommandCopyright, messages.AboutCommandLicense)
	// textAttributions := fmt.Sprintf("**%s**", messages.AboutCommandAttributions)
	// textFooter := fmt.Sprintf("%s\n%s", messages.AboutCommandHelp, messages.AboutCommandAwoo)

	// text := fmt.Sprintf("%s\n%s\n%s", textCopyright, textAttributions, textFooter)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: ascii,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.NavButton(messages.MyProfileLabel, discordgo.PrimaryButton, router.ViewMe),
					router.NavButton(messages.HelpLabel, discordgo.SecondaryButton, router.ViewHelp),
					discordgo.Button{
						Label: messages.AboutCommandGitHubLabel,
						Style: discordgo.LinkButton,
						URL:   messages.AboutCommandGitHubRepoLink,
					},
				}},
			},
		},
	})
}

// RenderAboutUpdate renders the about page as a message update (used by the router from /me).
func RenderAboutUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: ascii,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.NavButton(messages.MyProfileLabel, discordgo.PrimaryButton, router.ViewMe),
					router.NavButton(messages.HelpLabel, discordgo.SecondaryButton, router.ViewHelp),
					discordgo.Button{
						Label: messages.AboutCommandGitHubLabel,
						Style: discordgo.LinkButton,
						URL:   messages.AboutCommandGitHubRepoLink,
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

var ascii2 = "```" + `
*		*	*  * 		*	🌕*
	*	  *		*   *     *
┏┏ ┏━┃┏━┃┏━ ━┏┛┏━┃┏━┃┏━┛┏━┛┏━┃
┃┃┃┃ ┃┃ ┃┃ ┃ ┃ ┏┏┛┏━┃┃  ┏━┛┏┏┛
┛┛┛━━┛━━┛┛ ┛ ┛ ┛ ┛┛ ┛━━┛━━┛┛ ┛
*	   *awoo*	*	*      *
	🐺        *			*    *  
` +
	messages.AboutCommandBotDesc +
	"```"

var ascii = fmt.Sprintf(`
%s
%s
-# Released under the %s license.
-# %s

%s

-# %s
-# 🐺 %s 🌕
`, ascii2,
	messages.AboutCommandCopyright,
	messages.AboutCommandLicense,
	messages.BotVersion,
	messages.AboutCommandHelp,
	messages.AboutCommandAttributions,
	messages.AboutCommandAwoo,
)
