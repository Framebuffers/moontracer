package commands

import (
	"fmt"
	"log"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

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
	if registered {
		respond(s, i, messages.AlreadyRegisteredMessage)
		return
	}

	player := &models.Player{ID: userID}
	if err := db.Insert(r.db, player); err != nil {
		log.Printf("register: %s: %v", messages.RegistrationInsertError, err)
		respond(s, i, messages.RegistrationFailureMessage)
		return
	}

	respond(s, i, fmt.Sprintf(messages.RegistrationSuccessMessage, userID))
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
