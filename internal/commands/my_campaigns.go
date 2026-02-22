package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

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

	registered, err := isRegistered(p.db, userID)
	if err != nil {
		log.Printf("%s %v", messages.RegistrationCheckError, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	entries, err := models.GetPlayerCampaigns(p.db, userID)
	if err != nil {
		log.Printf("%s %s: %v", messages.PlayerFetchErrorMessage, userID, err)
		respond(s, i, messages.MyCampaignsLoadError)
		return
	}

	if len(entries) == 0 {
		respond(s, i, messages.NoCampaignsMessage)
		return
	}

	var buttons []discordgo.MessageComponent
	var lines []string
	for _, e := range entries {
		campaignName := e.CampaignID
		campaignTag := e.CampaignID
		if e.Campaign != nil {
			campaignName = e.Campaign.Name
			campaignTag = e.Campaign.Tag
		}
		lines = append(lines, fmt.Sprintf("**%s** — %s (%s)", campaignName, e.Role, e.Status))
		buttons = append(buttons, discordgo.Button{
			Label:    campaignName,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("campaign_view:%s", campaignTag),
		})
	}

	// Discord limits ActionsRow to 5 buttons
	var rows []discordgo.MessageComponent
	for idx := 0; idx < len(buttons); idx += 5 {
		end := min(idx+5, len(buttons))
		rows = append(rows, discordgo.ActionsRow{Components: buttons[idx:end]})
	}

	var content strings.Builder
	content.WriteString("Your campaigns:\n")
	for _, l := range lines {
		content.WriteString(l + "\n")
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content.String(),
			Components: rows,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
