package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
newSessionCommand opens the new-session modal for a DM's campaign.

Flow:
 1. DM types /newsession and selects a campaign via autocomplete.
 2. Bot opens the shared new-session modal.
 3. DM fills in date, time, and optional notes, then submits.
 4. The newSessionModal handler (interactions package) creates the session record,
    posts the channel announcement with RSVP buttons, and DMs all active members.
*/
type newSessionCommand struct {
	db *bun.DB
}

func (c *newSessionCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.NewSessionCommandName,
		Description: messages.NewSessionCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         messages.NewSessionOptionName,
				Description:  messages.NewSessionOptionDesc,
				Required:     true,
				Autocomplete: true,
			},
		},
	}
}

func (c *newSessionCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	ok, err := auth.Authorize(c.db, userID, auth.ScopeDM, "")
	if err != nil {
		log.Printf("newsession: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.ManageNotAuthorized)
		return
	}

	var campaignID string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.NewSessionOptionName {
			campaignID = opt.StringValue()
		}
	}

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil || campaign.DungeonMaster != userID || campaign.IsArchived {
		respond(s, i, messages.ManageCampaignNotFound)
		return
	}

	settings, _ := models.GetOrCreatePlayerSettings(c.db, userID)
	loc := settings.Location()
	timeLabel := fmt.Sprintf(messages.NewCampaignScheduleTimeLabelFmt, helpers.TZLabel(loc))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   fmt.Sprintf("%s:%s", messages.NewSessionModalID, campaignID),
			Title:      messages.NewSessionModalTitle,
			Components: NewSessionModalRows(timeLabel),
		},
	})
}

func (c *newSessionCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.NewSessionOptionName {
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

/*
NewSessionModalRows returns the standard modal component rows for session scheduling.
Exported so the manage-menu button handler can reuse it without duplication.
*/
func NewSessionModalRows(timeLabel string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.NewSessionDateFieldID,
				Label:       messages.NewCampaignScheduleDateLabel,
				Style:       discordgo.TextInputShort,
				Required:    true,
				Placeholder: messages.NewCampaignScheduleDatePlaceholder,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.NewSessionTimeFieldID,
				Label:       timeLabel,
				Style:       discordgo.TextInputShort,
				Required:    true,
				Placeholder: messages.NewCampaignScheduleTimePlaceholder,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.NewSessionNotesFieldID,
				Label:       messages.NewSessionNotesLabel,
				Style:       discordgo.TextInputParagraph,
				Required:    false,
				Placeholder: messages.NewSessionNotesPlaceholder,
				MaxLength:   500,
			},
		}},
	}
}
