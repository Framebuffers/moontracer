package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type newCampaign struct {
	db *bun.DB
}

func (n *newCampaign) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "newcampaign",
		Description: "Create a new campaign (you will be the DM).",
	}
}

func (n *newCampaign) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := isRegistered(n.db, userID)
	if err != nil {
		log.Printf("%s %v", messages.RegistrationCheckError, err)
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
			CustomID: "modal_campaign_create",
			Title:    "Create a New Campaign",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "description",
						Label:       "Description",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "Describe your campaign setting and premise...",
						Required:    true,
						MaxLength:   1000,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "edition",
						Label:       "Edition",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. 5e, 3.5e, PF2e",
						Required:    true,
						MaxLength:   20,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "rules",
						Label:       "Rules",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. 2024, 2014, homebrew",
						Required:    false,
						MaxLength:   50,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "slots",
						Label:       "Player Slots",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. 4",
						Required:    true,
						MaxLength:   3,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "warnings",
						Label:       "Content Warnings (comma-separated)",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. Violence, Horror, Permadeath",
						Required:    false,
						MaxLength:   200,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("%s %v", messages.NewCampaignModalError, err)
	}
}
