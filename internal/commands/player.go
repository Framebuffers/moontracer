package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type playerCommand struct {
	db *bun.DB
}

func (p *playerCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "mycampaigns",
		Description: "List the campaigns you're part of.",
	}
}

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
		lines = append(lines, fmt.Sprintf("**%s** — %s (%s)", e.CampaignID, e.Role, e.Status))
		buttons = append(buttons, discordgo.Button{
			Label:    e.CampaignID,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("campaign_view:%s", e.CampaignID),
		})
	}

	// Discord limits ActionsRow to 5 buttons
	var rows []discordgo.MessageComponent
	for idx := 0; idx < len(buttons); idx += 5 {
		end := idx + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons[idx:end]})
	}

	content := "Your campaigns:\n"
	for _, l := range lines {
		content += l + "\n"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: rows,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
