package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type manageCampaigns struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (m *manageCampaigns) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.ManageCampaignsCommandName,
		Description: messages.ManageCampaignsCommandDesc,
	}
}

// Execute lists all campaigns the invoker DMs as a select menu, plus a New Campaign button.
func (m *manageCampaigns) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(m.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("managecampaigns: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	entries, err := models.GetPlayerCampaigns(m.db, userID)
	if err != nil {
		log.Printf("managecampaigns: failed to load campaigns: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	var dmEntries []models.CampaignPlayer
	for _, e := range entries {
		if e.Role == models.RoleDM {
			dmEntries = append(dmEntries, e)
		}
	}

	if len(dmEntries) == 0 {
		respond(s, i, messages.ManageNoDMCampaigns)
		return
	}

	var options []discordgo.SelectMenuOption
	var lines []string
	for _, e := range dmEntries {
		if len(options) >= 25 {
			break
		}
		name := e.CampaignID
		if e.Campaign != nil {
			name = e.Campaign.Name
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       name,
			Value:       e.CampaignID,
			Description: fmt.Sprintf("%s — %s", e.Role, e.Status),
		})
		lines = append(lines, fmt.Sprintf("**%s** — %s", name, e.Status))
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    messages.ManageSelectPrefix,
		Placeholder: messages.ManageCampaignsPlaceholder,
		Options:     options,
	}

	content := messages.ManageCampaignsListHeader + strings.Join(lines, "\n")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.BackLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: messages.BackMeID,
						Emoji:    &discordgo.ComponentEmoji{Name: "◀"},
					},
					discordgo.Button{
						Label:    messages.NewCampaignLabel,
						Style:    discordgo.SuccessButton,
						CustomID: "stub_newcampaign",
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
