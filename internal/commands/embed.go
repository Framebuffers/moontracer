package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// CampaignEmbed builds a rich embed for displaying campaign details.
func CampaignEmbed(c models.Campaign, players []models.CampaignPlayer) *discordgo.MessageEmbed {
	status := messages.ClosedStatusLabel
	if c.IsOpen {
		status = messages.OpenStatusLabel
	}

	campaignType := messages.CampaignLabel
	if c.IsOneshot {
		campaignType = messages.CampaignTypeOneShotLabel
	}
	if c.IsWestmarch {
		campaignType = messages.CampaignTypeWestmarchLabel
	}

	var playerLines []string
	for _, p := range players {
		playerLines = append(playerLines, fmt.Sprintf("<@%s> — %s (%s, %d sessions)",
			p.PlayerID, p.Role, p.Status, p.SessionsPlayed))
	}
	playersValue := messages.NoneLabel
	if len(playerLines) > 0 {
		playersValue = strings.Join(playerLines, "\n")
	}

	warnings := messages.NoneLabel
	if len(c.Warnings) > 0 {
		warnings = strings.Join(c.Warnings, ", ")
	}

	books := messages.NoBooksSpecifiedLabel
	if len(c.Game.BooksAllowed) > 0 {
		books = strings.Join(c.Game.BooksAllowed, ", ")
	}

	slotsValue := "Unlimited"
	if c.Slots > 0 {
		slotsValue = fmt.Sprintf("%d", c.Slots)
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "DM", Value: fmt.Sprintf("<@%s>", c.DungeonMaster), Inline: true},
		{Name: "Status", Value: status, Inline: true},
		{Name: "Slots", Value: slotsValue, Inline: true},
		{Name: "Edition", Value: c.Game.Edition, Inline: true},
	}

	if c.Game.Rules != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Rules", Value: c.Game.Rules, Inline: true})
	}
	if c.Game.VTT != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "VTT", Value: c.Game.VTT, Inline: true})
	}

	fields = append(fields,
		&discordgo.MessageEmbedField{Name: "Books", Value: books, Inline: false},
		&discordgo.MessageEmbedField{Name: "Schedule", Value: FormatSchedule(c), Inline: false},
		&discordgo.MessageEmbedField{Name: "Warnings", Value: warnings, Inline: false},
		&discordgo.MessageEmbedField{Name: fmt.Sprintf("Players (%d)", len(players)), Value: playersValue, Inline: false},
	)

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s — %s", campaignType, c.Name),
		Description: c.Description,
		Color:       messages.EmbedColor,
		Fields:      fields,
	}
}

// CampaignButtons builds context-aware action buttons for a campaign view.
func CampaignButtons(callerID string, c models.Campaign, players []models.CampaignPlayer) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if c.DungeonMaster == callerID {
		buttons = append(buttons, discordgo.Button{
			Label:    messages.ManageCampaignButtonLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("manage_campaign:%s", c.ID),
		})
		return buttons
	}

	isCallerMember := false
	for _, p := range players {
		if p.PlayerID == callerID && p.Status == models.StatusActive {
			isCallerMember = true
			break
		}
	}

	if isCallerMember {
		buttons = append(buttons, discordgo.Button{
			Label:    messages.LeaveCampaignLabel,
			Style:    discordgo.DangerButton,
			CustomID: fmt.Sprintf("campaign_leave:%s", c.Tag),
		})
	} else if c.IsOpen {
		buttons = append(buttons, discordgo.Button{
			Label:    messages.JoinCampaignLabel,
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("campaign_join:%s", c.Tag),
		})
	}

	return buttons
}

// FormatSchedule builds a human-readable schedule string for the campaign embed.
func FormatSchedule(c models.Campaign) string {
	sched := c.Schedule

	if !sched.HasSchedule() {
		return fmt.Sprintf("%s — Schedule not set", sched.Frequency)
	}

	line := fmt.Sprintf("%s — %s %s UTC (%.0fh)", sched.Frequency, sched.DayName(), sched.StartTime, sched.DurationHours)

	if !sched.NextSession.IsZero() {
		line += fmt.Sprintf("\nNext: %s", sched.NextSession.Format("2006-01-02"))
	}
	if !sched.LastSession.IsZero() {
		line += fmt.Sprintf(" | Last: %s", sched.LastSession.Format("2006-01-02"))
	}

	return line
}
