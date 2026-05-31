package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/campaigns`.
		2. Load all approved, non-archived campaigns.
		3. Show format filter dropdown + campaign select dropdown.
*/

type campaignsCommand struct {
	db *bun.DB
}

func (c *campaignsCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.CampaignsCommandName,
		Description: messages.CampaignsCommandDesc,
	}
}

func (c *campaignsCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaigns, err := db.GetAll[models.Campaign](c.db)
	if err != nil {
		log.Printf("campaigns: failed to load campaigns: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	var filtered []models.Campaign
	for _, camp := range campaigns {
		if camp.IsApproved && !camp.IsArchived {
			filtered = append(filtered, camp)
		}
	}

	filterMenu := discordgo.SelectMenu{
		CustomID:    messages.CampaignsFilterPrefix,
		Placeholder: messages.CampaignsFilterPlaceholder,
		Options: []discordgo.SelectMenuOption{
			{Label: messages.CampaignsFilterAll, Value: "all", Default: true},
			{Label: messages.CampaignsFilterCampaign, Value: "campaign"},
			{Label: messages.CampaignsFilterOneshot, Value: "oneshot"},
			{Label: messages.CampaignsFilterWestmarch, Value: "westmarch"},
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{filterMenu}},
	}

	content := messages.CampaignsCommandDesc
	if len(filtered) > 0 {
		campaignSelect := buildCampaignSelectMenu(filtered, messages.CampaignSelectPrefix, messages.CampaignsSelectPlaceholder)
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{campaignSelect}})
	} else {
		content = messages.CampaignsNoneAvailable
	}

	components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
	}})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
buildCampaignSelectMenu builds a select menu from campaigns.

Note:

	In order to avoid an import cycle, this method is local.
*/
func buildCampaignSelectMenu(campaigns []models.Campaign, customID, placeholder string) discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, c := range campaigns {
		if len(options) >= 25 {
			break
		}

		format := "Campaign"
		if c.IsOneshot {
			format = "One-shot"
		}
		if c.IsWestmarch {
			format = "Westmarch"
		}
		status := "Open"
		if !c.IsOpen {
			status = "Closed"
		}
		slots := c.DisplaySlots()
		if c.Slots > 0 && c.Slots <= 10 {
			slots = fmt.Sprintf("%s slots", slots)
		}
		desc := fmt.Sprintf("%s- %s, %s", format, status, slots)

		options = append(options, discordgo.SelectMenuOption{
			Label:       c.Name,
			Value:       c.ID,
			Description: desc,
		})
	}

	if len(options) == 0 {
		options = append(options, discordgo.SelectMenuOption{
			Label:   "No campaigns available",
			Value:   "none",
			Default: true,
		})
	}

	return discordgo.SelectMenu{
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}
