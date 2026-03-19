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

// Execute lists all campaigns the invoker DMs and shows selection buttons.
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

	var buttons []discordgo.MessageComponent
	var lines []string
	for _, e := range dmEntries {
		campaignName := e.CampaignID
		if e.Campaign != nil {
			campaignName = e.Campaign.Name
		}
		lines = append(lines, fmt.Sprintf("**%s** — %s", campaignName, e.Status))
		buttons = append(buttons, discordgo.Button{
			Label:    campaignName,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("manage_campaign:%s", e.CampaignID),
		})
	}

	// info: discord limits ActionsRow to 5 buttons.
	var rows []discordgo.MessageComponent
	for idx := 0; idx < len(buttons); idx += 5 {
		end := min(idx+5, len(buttons))
		rows = append(rows, discordgo.ActionsRow{Components: buttons[idx:end]})
	}

	var content strings.Builder
	content.WriteString("Your campaigns (DM):\n")
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
