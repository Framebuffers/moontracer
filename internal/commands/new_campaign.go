package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/messages"
)

// newCampaign creates a modal to input information needed to create a new Campaign.
type newCampaign struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (n *newCampaign) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.NewCampaignCommandName,
		Description: messages.NewCampaignCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (n *newCampaign) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(n.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("new_campaign: %s: %v", messages.RegistrationCheckError, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.NewCampaignModalCustomID,
			Title:    messages.NewCampaignModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldNameID,
						Label:       messages.FieldNameLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldNamePlaceholder,
						Required:    true,
						MaxLength:   100,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldTagID,
						Label:       messages.FieldTagLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldTagPlaceholder,
						Required:    true,
						MaxLength:   30,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldDescriptionID,
						Label:       messages.FieldDescriptionLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.FieldDescriptionPlaceholder,
						Required:    true,
						MaxLength:   1000,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldEditionID,
						Label:       messages.FieldEditionLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldEditionPlaceholder,
						Required:    true,
						MaxLength:   20,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldSlotsID,
						Label:       messages.FieldSlotsLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldSlotsPlaceholder,
						Required:    true,
						MaxLength:   3,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("new_campaign: %s: %v", messages.NewCampaignModalError, err)
	}
}
