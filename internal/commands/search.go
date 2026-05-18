package commands

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. User types `/search name:<query>`.
		2. Discord sends autocomplete interactions as the user types.
		3. Autocomplete handler returns matching campaign names.
		4. User selects a campaign from the autocomplete list.
		5. Execute renders the campaign detail embed.
*/

type searchCommand struct {
	db *bun.DB
}

func (c *searchCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.SearchCommandName,
		Description: messages.SearchCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         messages.SearchOptionName,
				Description:  messages.SearchOptionDesc,
				Required:     true,
				Autocomplete: true,
			},
		},
	}
}

func (c *searchCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var campaignID string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.SearchOptionName {
			campaignID = opt.StringValue()
		}
	}

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil || !campaign.IsApproved {
		respond(s, i, messages.SearchNoResults)
		return
	}

	players, err := models.GetCampaignPlayers(c.db, campaign.ID)
	if err != nil {
		log.Printf("search: failed to load players for %s: %v", campaignID, err)
		respond(s, i, messages.CampaignPlayersLoadError)
		return
	}

	userID := i.Member.User.ID
	coverURL := models.CoverURLForCampaign(c.db, campaign.ID)
	embed := CampaignEmbed(*campaign, players, coverURL, "", userID)
	buttons := CampaignButtons(userID, *campaign, players, "")

	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	}

	if len(buttons) > 0 {
		resp.Data.Components = []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: buttons},
		}
	}

	s.InteractionRespond(i.Interaction, resp)
}

func (c *searchCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.SearchOptionName {
			query = opt.StringValue()
		}
	}

	campaigns, err := db.GetAll[models.Campaign](c.db)
	if err != nil {
		log.Printf("search autocomplete: failed to load campaigns: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{},
		})
		return
	}

	queryLower := strings.ToLower(query)
	var choices []*discordgo.ApplicationCommandOptionChoice

	for _, camp := range campaigns {
		if !camp.IsApproved || camp.IsArchived {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(camp.Name), queryLower) {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  camp.Name,
			Value: camp.ID,
		})
		if len(choices) >= 25 {
			break
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}
