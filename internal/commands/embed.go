package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
CampaignEmbed builds a rich embed for displaying campaign details.

coverURL is shown as the main image. viewerTokenURL, if set, is shown as a
thumbnail (top-right).

This is used to display the viewing player's character token.
*/
func CampaignEmbed(c models.Campaign, players []models.CampaignPlayer, coverURL, viewerTokenURL, callerID string) *discordgo.MessageEmbed {
	status := messages.ClosedStatusLabel
	if c.IsArchived {
		status = messages.ArchivedStatusLabel
	} else if c.IsOpen {
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
		if p.Status == models.StatusBanned {
			continue
		}
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

	slotsValue := c.DisplaySlots()

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
	if c.ChannelID != "" && callerIsMember(callerID, c, players) {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Channel", Value: fmt.Sprintf("<#%s>", c.ChannelID), Inline: true})
	}

	synopsis := c.Description
	if synopsis == "" {
		synopsis = messages.NoneLabel
	}
	fields = append(fields,
		&discordgo.MessageEmbedField{Name: "Synopsis", Value: synopsis, Inline: false},
		&discordgo.MessageEmbedField{Name: "Schedule", Value: FormatSchedule(c), Inline: false},
		&discordgo.MessageEmbedField{Name: "Warnings", Value: warnings, Inline: false},
		&discordgo.MessageEmbedField{Name: fmt.Sprintf("Players (%d)", len(playerLines)), Value: playersValue, Inline: false},
	)

	if links := formatEmbedLinks(c); links != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: messages.ManageLinksEmbedTitle, Value: links, Inline: false})
	}

	embed := &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s — %s", campaignType, c.Name),
		Color:  messages.EmbedColor,
		Fields: fields,
	}
	if coverURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: coverURL}
	}
	if viewerTokenURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: viewerTokenURL}
	}
	return embed
}

/*
CampaignButtons builds context-aware action rows for a campaign view.

viewerSheetURL is the viewing player's own sheet URL; when set, an "Open Sheet"
link button is prepended to their action row.

Pass "" for non-members.

Returns complete ActionsRows ready to append directly to a component list.
*/
func CampaignButtons(callerID string, c models.Campaign, players []models.CampaignPlayer, viewerSheetURL string) []discordgo.MessageComponent {
	if c.IsArchived {
		return nil
	}

	if c.DungeonMaster == callerID {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.ManageCampaignButtonLabel,
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("manage_campaign:%s", c.ID),
				},
			}},
		}
	}

	isCallerMember := false
	for _, p := range players {
		if p.PlayerID == callerID && p.Status == models.StatusActive {
			isCallerMember = true
			break
		}
	}

	if isCallerMember {
		row1 := []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.PlayerSetSheetLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.PlayerSetSheetPrefix, c.ID),
			},
			discordgo.Button{
				Label:    messages.PlayerSetTokenLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.PlayerSetTokenPrefix, c.ID),
			},
			discordgo.Button{
				Label:    messages.PlayerContactDMLabel,
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("%s:%s", messages.PlayerContactDMPrefix, c.ID),
			},
		}
		if viewerSheetURL != "" {
			row1 = append([]discordgo.MessageComponent{
				discordgo.Button{
					Label: messages.PlayerOpenSheetLabel,
					Style: discordgo.LinkButton,
					URL:   viewerSheetURL,
				},
			}, row1...)
		}
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: row1},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.LeaveCampaignLabel,
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("%s:%s", messages.PlayerLeaveConfirmPrefix, c.ID),
				},
			}},
		}
	}

	if c.IsOpen {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.JoinCampaignLabel,
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("campaign_join:%s", c.Tag),
				},
			}},
		}
	}

	return nil
}

// callerIsMember returns true if callerID is the DM or an active player of the campaign.
func callerIsMember(callerID string, c models.Campaign, players []models.CampaignPlayer) bool {
	if callerID == "" {
		return false
	}
	if c.DungeonMaster == callerID {
		return true
	}
	for _, p := range players {
		if p.PlayerID == callerID && p.Status == models.StatusActive {
			return true
		}
	}
	return false
}

func formatEmbedLinks(c models.Campaign) string {
	var parts []string
	if c.VTTLink != "" {
		parts = append(parts, fmt.Sprintf("**VTT:** %s", c.VTTLink))
	}
	for _, r := range c.Links {
		parts = append(parts, fmt.Sprintf("• %s", r))
	}
	return strings.Join(parts, "\n")
}

// FormatSchedule builds a human-readable schedule string for the campaign embed.
func FormatSchedule(c models.Campaign) string {
	sched := c.Schedule

	if !sched.HasSchedule() {
		if sched.Frequency == "" {
			return "Schedule not set"
		}
		return fmt.Sprintf("%s — schedule not set", sched.Frequency)
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
