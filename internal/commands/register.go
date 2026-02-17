package commands

import (
	"fmt"
	"log"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type registerCommand struct {
	db *bun.DB
}

func (r *registerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Register as a player so you can join and create campaigns.",
	}
}

func (r *registerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := isRegistered(r.db, userID)
	if err != nil {
		log.Printf("%s %v", messages.RegistrationCheckError, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if registered {
		respond(s, i, messages.AlreadyRegisteredMessage)
		return
	}

	player := &models.Player{ID: userID}
	if err := db.Insert(r.db, player); err != nil {
		log.Printf("%s %v", messages.RegistrationInsertError, err)
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
