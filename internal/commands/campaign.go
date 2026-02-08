package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/manager/models"
)

type campaignCommand struct{}

func (c *campaignCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "campaign",
		Description: "Show campaign details (mock data)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "id",
				Description: "Campaign ID to look up",
				Required:    false,
			},
		},
	}
}

func (c *campaignCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Read optional ID (ignored for now, always returns mock data)
	var id string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "id" {
			id = opt.StringValue()
		}
	}
	if id == "" {
		id = "moontracer-mock-campaign"
	}

	campaign, players := mockCampaign(id)
	embed := campaignEmbed(campaign, players)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func mockCampaign(id string) (models.Campaign, []models.CampaignPlayer) {
	campaign := models.Campaign{
		ID:            id,
		DungeonMaster: "123456789",
		Description:   "A fantasy about a dog howling at the moon. Be careful, he can be (very) soft.",
		Game: models.GameConfig{
			Edition:      "5e",
			Rules:        "2024",
			VTT:          "owlbear-legacy",
			BooksAllowed: []string{"PHB", "DMG", "MM", "XGE", "TCE"},
		},
		Slots:     5,
		IsOpen:    true,
		IsOneshot: false,
		Warnings:  []string{"Softness Alert", "Permadeath"},
		Schedule: models.CampaignSchedule{
			Frequency:   models.Weekly,
			CreatedAt:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			LastSession: time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	players := []models.CampaignPlayer{
		{PlayerID: "123456789", CampaignID: id, Role: models.RoleDM, Status: models.StatusActive, SessionsPlayed: 4},
		{PlayerID: "987654321", CampaignID: id, Role: models.RolePlayer, Status: models.StatusActive, SessionsPlayed: 4},
		{PlayerID: "111222333", CampaignID: id, Role: models.RolePlayer, Status: models.StatusHiatus, SessionsPlayed: 2},
	}

	return campaign, players
}

func campaignEmbed(c models.Campaign, players []models.CampaignPlayer) *discordgo.MessageEmbed {
	status := "Closed"
	if c.IsOpen {
		status = "Open"
	}

	campaignType := "Campaign"
	if c.IsOneshot {
		campaignType = "One-shot"
	}

	// build the campaign:
	// 1. add players list
	var playerLines []string
	for _, p := range players {
		playerLines = append(playerLines, fmt.Sprintf("<@%s> — %s (%s, %d sessions)",
			p.PlayerID, p.Role, p.Status, p.SessionsPlayed))
	}

	// 2. add warnings
	warnings := "None"
	if len(c.Warnings) > 0 {
		warnings = strings.Join(c.Warnings, ", ")
	}

	// 3. add books
	books := "None specified"
	if len(c.Game.BooksAllowed) > 0 {
		books = strings.Join(c.Game.BooksAllowed, ", ")
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s — %s", campaignType, c.ID),
		Description: c.Description,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "DM", Value: fmt.Sprintf("<@%s>", c.DungeonMaster), Inline: true},
			{Name: "Status", Value: status, Inline: true},
			{Name: "Slots", Value: fmt.Sprintf("%d", c.Slots), Inline: true},
			{Name: "Edition", Value: c.Game.Edition, Inline: true},
			{Name: "Rules", Value: c.Game.Rules, Inline: true},
			{Name: "VTT", Value: c.Game.VTT, Inline: true},
			{Name: "Books", Value: books, Inline: false},
			{Name: "Schedule", Value: fmt.Sprintf("%s (last session: %s)",
				c.Schedule.Frequency, c.Schedule.LastSession.Format("2006-01-02")), Inline: false},
			{Name: "Warnings", Value: warnings, Inline: false},
			{Name: fmt.Sprintf("Players (%d)", len(players)), Value: strings.Join(playerLines, "\n"), Inline: false},
		},
	}
}

func getScheduledCampaigns(c models.Player) (*models.Campaign, error) {
	var schedule models.CampaignSchedule

}
