package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
manageCommand is a shortcut to the campaign management hub.

Flow:
 1. DM types /manage and selects a campaign via autocomplete (only their non-archived DM campaigns appear).
 2. Bot opens the campaign management menu directly, without needing the /me hub as a waypoint.
*/
type manageCommand struct {
	db *bun.DB
}

func (c *manageCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.ManageCommandName,
		Description: messages.ManageCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         messages.ManageCommandOptionName,
				Description:  messages.ManageCommandOptionDesc,
				Required:     true,
				Autocomplete: true,
			},
		},
	}
}

func (c *manageCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	var campaignID string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.ManageCommandOptionName {
			campaignID = opt.StringValue()
		}
	}

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil || campaign.DungeonMaster != userID || campaign.IsArchived {
		respond(s, i, messages.ManageCampaignNotFound)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.NavButton(messages.ManagePlayersLabel, discordgo.PrimaryButton, router.ViewManagePlayers, campaignID),
					router.NavButton(messages.ManageSessionsLabel, discordgo.PrimaryButton, router.ViewManageSessions, campaignID),
					router.NavButton(messages.ManageSettingsLabel, discordgo.PrimaryButton, router.ViewManageSettings, campaignID),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *manageCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.ManageCommandOptionName {
			query = opt.StringValue()
		}
	}

	ctx := context.Background()
	var players []models.CampaignPlayer
	_ = c.db.NewSelect().Model(&players).
		Relation("Campaign").
		Where("campaign_player.player_id = ? AND campaign_player.role = ?", userID, models.RoleDM).
		Scan(ctx)

	queryLower := strings.ToLower(query)
	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, p := range players {
		if p.Campaign == nil || !p.Campaign.IsApproved || p.Campaign.IsArchived {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(p.Campaign.Name), queryLower) {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  p.Campaign.Name,
			Value: p.Campaign.ID,
		})
		if len(choices) >= 25 {
			break
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
