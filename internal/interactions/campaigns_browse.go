package interactions

import (
	"moontracer/internal/interactions/helpers"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// RenderCampaignsBrowse renders the campaign browse view with filter + select menu.
func RenderCampaignsBrowse(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, filter string) {
	campaigns, err := db.GetAll[models.Campaign](database)
	if err != nil {
		log.Printf("campaigns_browse: failed to load campaigns: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	var filtered []models.Campaign
	for _, c := range campaigns {
		if !c.IsApproved || c.IsArchived {
			continue
		}
		switch filter {
		case "oneshot":
			if !c.IsOneshot {
				continue
			}
		case "westmarch":
			if !c.IsWestmarch {
				continue
			}
		case "campaign":
			if c.IsOneshot || c.IsWestmarch {
				continue
			}
		}
		filtered = append(filtered, c)
	}

	filterMenu := discordgo.SelectMenu{
		CustomID:    messages.CampaignsFilterPrefix,
		Placeholder: messages.CampaignsFilterPlaceholder,
		Options: []discordgo.SelectMenuOption{
			{Label: messages.CampaignsFilterAll, Value: "all", Default: filter == "all"},
			{Label: messages.CampaignsFilterCampaign, Value: "campaign", Default: filter == "campaign"},
			{Label: messages.CampaignsFilterOneshot, Value: "oneshot", Default: filter == "oneshot"},
			{Label: messages.CampaignsFilterWestmarch, Value: "westmarch", Default: filter == "westmarch"},
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{filterMenu}},
	}

	content := messages.CampaignsCommandDesc
	if len(filtered) > 0 {
		campaignSelect := BuildCampaignSelect(filtered, messages.CampaignSelectPrefix, messages.CampaignsSelectPlaceholder)
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{campaignSelect}})
	} else {
		content = messages.CampaignsNoneAvailable
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     []*discordgo.MessageEmbed{},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// campaignsFilterHandler handles the format filter dropdown selection.
type campaignsFilterHandler struct {
	db *bun.DB
}

func (h *campaignsFilterHandler) CustomIDPrefix() string {
	return messages.CampaignsFilterPrefix
}

func (h *campaignsFilterHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	filter := "all"
	if len(values) > 0 {
		filter = values[0]
	}
	RenderCampaignsBrowse(s, i, h.db, filter)
}

// campaignSelectHandler handles when user picks a campaign from the browse dropdown.
type campaignSelectHandler struct {
	db *bun.DB
}

func (h *campaignSelectHandler) CustomIDPrefix() string {
	return messages.CampaignSelectPrefix
}

func (h *campaignSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	if len(values) == 0 || values[0] == "none" {
		return
	}
	campaignID := values[0]
	RenderCampaignDetail(s, i, h.db, campaignID)
}
