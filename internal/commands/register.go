package commands

import (
	"fmt"
	"log"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

/*
	Flow:
		1. User runs `/register`.
		2. Check if the user is already registered (ScopePlayer auth check).
		3. If already registered, respond: "You are already registered!"
		4. If not, insert a new Player record into the DB with the user's Discord ID.
		5. Respond: "Welcome, @user! You are now registered."
*/

type registerCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (r *registerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.RegisterCommandName,
		Description: messages.RegisterCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (r *registerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(r.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("register: %s: %v", messages.RegistrationCheckError, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	content := messages.AlreadyRegisteredMessage
	if !registered {
		player := &models.Player{ID: userID}
		if err := db.Insert(r.db, player); err != nil {
			log.Printf("register: %s: %v", messages.RegistrationInsertError, err)
			respond(s, i, messages.RegistrationFailureMessage)
			return
		}
		content = fmt.Sprintf(messages.RegistrationSuccessMessage, userID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content + "\n\nWhat would you like to do?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.BrowseCampaignsLabel,
						Style:    discordgo.PrimaryButton,
						CustomID: router.NavCustomID(router.ViewCampaignsBrowse, "all"),
					},
					discordgo.Button{
						Label:    messages.MyProfileLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: router.NavCustomID(router.ViewMe),
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// respond is a helper to send an ephemeral text response.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
respondNotRegistered sends a fresh ephemeral with the "not registered" message and a
Register button that registers the user and opens the /me hub in one click.
*/
func respondNotRegistered(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messages.NotRegisteredMessage,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.RegisterButtonLabel,
						Style:    discordgo.SuccessButton,
						CustomID: messages.QuickRegisterPrefix,
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
