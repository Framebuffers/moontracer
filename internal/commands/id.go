package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// campaignCommand returns an embed with the details of a Campaign.
type campaignCommand struct {
	db *bun.DB
}

func (c *campaignCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.CampaignCommandName,
		Description: messages.CampaignCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.TagCommandName,
				Description: messages.TagCommandDesc,
				Required:    true,
			},
		},
	}
}

func (c *campaignCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var tag string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.TagCommandName {
			tag = opt.StringValue()
		}
	}

	campaign, err := db.GetByTag[models.Campaign](c.db, tag)
	if err != nil {
		log.Printf(messages.CampaignFetchError+"%v", tag, err)
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(c.db, campaign.ID)
	if err != nil {
		log.Printf("%s %s: %v", messages.PlayerFetchErrorMessage, tag, err)
		respond(s, i, messages.CampaignPlayersLoadError)
		return
	}

	embed := campaignEmbed(*campaign, players)
	buttons := campaignButtons(i.Member.User.ID, *campaign, players)

	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	}

	if len(buttons) > 0 {
		resp.Data.Components = []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: buttons},
		}
	}

	s.InteractionRespond(i.Interaction, resp)
}

func campaignButtons(callerID string, c models.Campaign, players []models.CampaignPlayer) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if c.DungeonMaster == callerID {
		toggleLabel := messages.OpenCampaignLabel
		if c.IsOpen {
			toggleLabel = messages.ClosedCampaignLabel
		}
		buttons = append(buttons,
			discordgo.Button{
				Label:    toggleLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("campaign_toggle:%s", c.Tag),
			},
		)
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

func campaignEmbed(c models.Campaign, players []models.CampaignPlayer) *discordgo.MessageEmbed {
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

	fields := []*discordgo.MessageEmbedField{
		{Name: "DM", Value: fmt.Sprintf("<@%s>", c.DungeonMaster), Inline: true},
		{Name: "Status", Value: status, Inline: true},
		{Name: "Slots", Value: fmt.Sprintf("%d", c.Slots), Inline: true},
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
		&discordgo.MessageEmbedField{Name: "Schedule", Value: fmt.Sprintf("%s (last session: %s)",
			c.Schedule.Frequency, c.Schedule.LastSession.Format("2006-01-02")), Inline: false},
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
