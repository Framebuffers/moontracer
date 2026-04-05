package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/me`.
		2. Authorize: check if registered.
		3. Show player hub with action buttons: My Campaigns, Next Sessions (stub), Notifications (stub).
*/

type meCommand struct {
	db *bun.DB
}

func (m *meCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.MeCommandName,
		Description: messages.MeCommandDesc,
	}
}

func (m *meCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(m.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("me: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.MeHubMessage, userID),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.MyCampaignsLabel,
						Style:    discordgo.PrimaryButton,
						CustomID: messages.BackMyCampaignsID,
					},
					discordgo.Button{
						Label:    messages.NextSessionsLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: "stub_nextsessions",
					},
					discordgo.Button{
						Label:    messages.NotificationsLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: "stub_notifications",
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// RenderMeHub is the shared rendering logic, callable from back buttons.
func RenderMeHub(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.MeHubMessage, userID),
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.MyCampaignsLabel,
						Style:    discordgo.PrimaryButton,
						CustomID: messages.BackMyCampaignsID,
					},
					discordgo.Button{
						Label:    messages.NextSessionsLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: "stub_nextsessions",
					},
					discordgo.Button{
						Label:    messages.NotificationsLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: "stub_notifications",
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
