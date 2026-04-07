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

/*
	Flow:
		1. User runs `/mycampaigns`.
		2. Authorize: check if the user is a registered player. Reject if not.
		3. Load all CampaignPlayer entries for this user (with Campaign relation).
		4. Render an ephemeral select menu (mycampaign_select) listing approved campaigns.
		5. Selecting an entry routes through myCampaignSelectHandler → RenderCampaignDetail.
*/

// playerCommand returns available information for a given player, like campaigns.
type playerCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (p *playerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.MyCampaignsCommandName,
		Description: messages.MyCampaignsCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (p *playerCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(p.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("my_campaigns: %s: %v", messages.RegistrationCheckError, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	entries, err := models.GetPlayerCampaigns(p.db, userID)
	if err != nil {
		log.Printf("my_campaigns: %s %s: %v", messages.PlayerFetchErrorMessage, userID, err)
		respond(s, i, messages.MyCampaignsLoadError)
		return
	}

	var approved []models.CampaignPlayer
	for _, e := range entries {
		if e.Campaign != nil && e.Campaign.IsApproved {
			approved = append(approved, e)
		}
	}

	if len(approved) == 0 {
		respond(s, i, messages.NoCampaignsMessage)
		return
	}

	var options []discordgo.SelectMenuOption
	var lines []string
	for _, e := range approved {
		if len(options) >= 25 {
			break
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       e.Campaign.Name,
			Value:       e.CampaignID,
			Description: fmt.Sprintf("%s — %s", e.Role, e.Status),
		})
		lines = append(lines, fmt.Sprintf("**%s** — %s (%s)", e.Campaign.Name, e.Role, e.Status))
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    messages.MyCampaignSelectPrefix,
		Placeholder: messages.MyCampaignsPlaceholder,
		Options:     options,
	}

	content := messages.MyCampaignsListHeader + strings.Join(lines, "\n")

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
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
