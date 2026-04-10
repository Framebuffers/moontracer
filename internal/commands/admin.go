package commands

/*
	Flow:
		1. User runs `/admin`.
		2. Authorize: check if the user is a mod or admin.
		3. Show admin hub with action buttons: Manage Campaigns, Active Campaigns (stub), Broadcast (stub), Database, Settings (stub).
*/

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/messages"
)

type adminCommand struct {
	db *bun.DB
}

func (a *adminCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AdminCommandName,
		Description: messages.AdminCommandDesc,
	}
}

func (a *adminCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	ok, err := auth.Authorize(a.db, userID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("admin: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AdminNotStaff)
		return
	}

	RenderAdminHub(s, i)
}

// RenderAdminHub renders the admin panel, callable from the slash command and back buttons.
func RenderAdminHub(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: adminHubData(),
	})
}

// RenderAdminHubUpdate re-renders the admin panel as a message update (for back navigation).
func RenderAdminHubUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: adminHubData(),
	})
}

func adminHubData() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Content: messages.AdminHubMessage,
		Embeds:  []*discordgo.MessageEmbed{},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.ManageCampaignsCommandDesc,
					Style:    discordgo.PrimaryButton,
					CustomID: messages.BackManageID,
				},
				discordgo.Button{
					Label:    messages.AdminCampaignsLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminCampaignsPrefix,
				},
				discordgo.Button{
					Label:    messages.AdminBroadcastLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminBroadcastPrefix,
				},
			}},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.AdminDatabaseLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminDatabasePrefix,
				},
				discordgo.Button{
					Label:    messages.AdminSettingsLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: messages.AdminSettingsPrefix,
				},
			}},
		},
		Flags: discordgo.MessageFlagsEphemeral,
	}
}
