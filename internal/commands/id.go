package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/campaign tag:<tag>`.
		2. Fetch the campaign by tag from the DB.
		3. Approval gate: if the campaign is not approved, respond with "not found".
		4. Load campaign players.
		5. Build a rich embed with campaign details (DM, status, slots, edition, schedule, players).
		6. Build context-aware buttons: Join (if open and not a member), Leave (if a member). DMs see no buttons here (they use /managecampaigns).
		7. Respond with the embed and buttons.
*/

// campaignCommand returns an embed with the details of a Campaign.
type campaignCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
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

// Execute is the logic that runs when the user invokes that command.
func (c *campaignCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var tag string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == messages.TagCommandName {
			tag = opt.StringValue()
		}
	}

	campaign, err := db.GetByTag[models.Campaign](c.db, tag)
	if err != nil {
		log.Printf("campaign: error fetching campaign %s: %v", tag, err)
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		respond(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(c.db, campaign.ID)
	if err != nil {
		log.Printf("campaign: %s %s: %v", messages.PlayerFetchErrorMessage, tag, err)
		respond(s, i, messages.CampaignPlayersLoadError)
		return
	}

	embed := CampaignEmbed(*campaign, players)
	buttons := CampaignButtons(i.Member.User.ID, *campaign, players)

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

