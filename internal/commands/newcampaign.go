package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/messages"
)

// newCampaignCommand opens the campaign creation modal directly as a slash command.
type newCampaignCommand struct {
	db *bun.DB
}

func (c *newCampaignCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.NewCampaignCommandName,
		Description: messages.NewCampaignCommandDesc,
	}
}

func (c *newCampaignCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	ok, err := auth.Authorize(c.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("newcampaign: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respondNotRegistered(s, i)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
						Placeholder: messages.FieldSlotsPlaceholder,
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
}
