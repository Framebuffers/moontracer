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
	"moontracer/internal/interactions/cdn"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
campaignUploadCommand implements /campaignupload.

Flow:

	1. User runs `/campaignupload kind:Cover campaign:<id> image:<file>`.
	2. Autocomplete on `campaign` surfaces campaigns the user DMs.
	3. Execute authorizes the caller as DM, validates the attachment,
	   stores the Discord CDN reference on the campaign, nulls the cache.
*/
type campaignUploadCommand struct {
	db *bun.DB
}

const maxCoverBytes = 8 * 1024 * 1024

func (c *campaignUploadCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.CampaignUploadCommandName,
		Description: messages.CampaignUploadCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        messages.CampaignUploadKindOptName,
				Description: messages.CampaignUploadKindOptDesc,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: messages.CampaignUploadKindCoverChoice, Value: "cover"},
				},
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         messages.CampaignUploadCampaignOptName,
				Description:  messages.CampaignUploadCampaignOptDesc,
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        messages.CampaignUploadImageOptName,
				Description: messages.CampaignUploadImageOptDesc,
				Required:    true,
			},
		},
	}
}

func (c *campaignUploadCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	var campaignID, attachmentID string
	for _, opt := range data.Options {
		switch opt.Name {
		case messages.CampaignUploadCampaignOptName:
			campaignID = opt.StringValue()
		case messages.CampaignUploadImageOptName:
			attachmentID = opt.Value.(string)
		}
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	campaign, err := db.GetByID[models.Campaign](c.db, campaignID)
	if err != nil {
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}
	if !campaign.CanMutate() {
		respond(s, i, messages.CampaignArchivedMessage)
		return
	}

	ok, err := auth.Authorize(c.db, userID, auth.ScopeDM, campaign.ID)
	if err != nil || !ok {
		respond(s, i, messages.CampaignUploadNotDM)
		return
	}

	att, ok := data.Resolved.Attachments[attachmentID]
	if !ok || att == nil {
		respond(s, i, messages.CampaignUploadMissingAttach)
		return
	}

	if !strings.HasPrefix(att.ContentType, "image/") {
		respond(s, i, messages.CampaignUploadNotImage)
		return
	}
	if att.Size > maxCoverBytes {
		respond(s, i, messages.CampaignUploadTooLarge)
		return
	}

	if err := cdn.SetCover(context.Background(), c.db, campaign, att.URL); err != nil {
		log.Printf("campaignupload: failed to save cover for %s: %v", campaign.ID, err)
		respond(s, i, messages.CampaignUploadFailure)
		return
	}

	respond(s, i, fmt.Sprintf(messages.CampaignUploadSuccess, campaign.Name))
}

func (c *campaignUploadCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	var query string
	for _, opt := range data.Options {
		if opt.Name == messages.CampaignUploadCampaignOptName && opt.Focused {
			query = opt.StringValue()
		}
	}

	campaigns, err := db.GetAll[models.Campaign](c.db)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{},
		})
		return
	}

	queryLower := strings.ToLower(query)
	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, camp := range campaigns {
		if camp.IsArchived || camp.DungeonMaster != userID {
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
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
