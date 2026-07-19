package commands

import (
	"fmt"
	"runtime"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/messages"
)

type aboutCommand struct{}

func (a *aboutCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AboutCommandName,
		Description: messages.AboutCommandDesc,
	}
}

func (a *aboutCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

var ascii2 = "```" + `
*		*	*  * 		*	🌕*
	*	  *		*   *     *
┏┏ ┏━┃┏━┃┏━ ━┏┛┏━┃┏━┃┏━┛┏━┛┏━┃
┃┃┃┃ ┃┃ ┃┃ ┃ ┃ ┏┏┛┏━┃┃  ┏━┛┏┏┛
┛┛┛━━┛━━┛┛ ┛ ┛ ┛ ┛┛ ┛━━┛━━┛┛ ┛
*  ┏┏ ┃ ┃━┏┛┃ ┃┏━┃┛┃    	*
*━┛┃┃┃━┏┛ ┃ ┏━┃┏┏┛┃┃  ━┛  *
*  ┛┛┛ ┛  ┛ ┛ ┛┛ ┛┛━━┛  	*
-- dev version --
*	   *awoo*	*	*      *
	🐺        *			*    *
` +
	messages.AboutCommandBotDesc +
	"```"

var ascii = fmt.Sprintf(`
%s

%s

-# %s
-# Released under the %s license.
-# %s, go: %s, discordgo: %s

-# %s
-# 🐺 %s 🌕
`, ascii2,
	messages.AboutCommandHelp,
	messages.AboutCommandCopyright,
	messages.AboutCommandLicense,
	messages.BotVersion,
	runtime.Version(),
	discordgo.VERSION,
	messages.AboutCommandAttributions,
	messages.AboutCommandAwoo,
)
