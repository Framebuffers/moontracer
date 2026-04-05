package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
RenderCampaignDetail shows a campaign embed with context-aware buttons.

Uses InteractionResponseUpdateMessage so it replaces the current message.
*/
func RenderCampaignDetail(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		log.Printf("campaign_detail: campaign %s not found: %v", campaignID, err)
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(database, campaign.ID)
	if err != nil {
		log.Printf("campaign_detail: failed to load players for %s: %v", campaignID, err)
		respondInteraction(s, i, messages.CampaignPlayersLoadError)
		return
	}

	userID := getUserID(i)
	embed := commands.CampaignEmbed(*campaign, players)
	actionButtons := commands.CampaignButtons(userID, *campaign, players)

	var components []discordgo.MessageComponent
	if len(actionButtons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: actionButtons})
	}

	components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		backButton(messages.BackLabel, messages.BackCampaignsID),
	}})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
