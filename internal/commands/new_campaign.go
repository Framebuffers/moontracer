package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/newcampaign`.
		2. Authorize: check if registered.
		3. Open a 3-field modal: Name, Max Players (optional), Synopsis & Rules.
		4. Submission is routed to `modal_campaign_create`, which creates the
		   campaign (pending) and shows the book/format config dropdowns.
		5. User picks game system + format, then clicks Submit for Approval,
		   which sends approval DMs to staff (handled in newcampaign_config.go).
*/

type newCampaign struct {
	db *bun.DB
}

func (n *newCampaign) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.NewCampaignCommandName,
		Description: messages.NewCampaignCommandDesc,
	}
}

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
						CustomID:    messages.FieldSlotsID,
						Label:       messages.FieldSlotsLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.FieldSlotsPlaceholderNew,
						Required:    false,
						MaxLength:   3,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.FieldDescriptionID,
						Label:       messages.FieldSynopsisLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.FieldDescriptionPlaceholder,
						Required:    true,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("new_campaign: %s: %v", messages.NewCampaignModalError, err)
	}
}
