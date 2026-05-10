package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
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
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved && !campaign.IsArchived {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(database, campaign.ID)
	if err != nil {
		log.Printf("campaign_detail: failed to load players for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignPlayersLoadError)
		return
	}

	userID := helpers.GetUserID(i)
	coverURL := models.CoverURLForCampaign(database, campaign.ID)

	var viewerTokenURL, viewerSheetURL string
	if cp, err := models.GetCampaignPlayer(database, userID, campaign.ID); err == nil {
		viewerSheetURL = cp.SheetURL
		if cp.MediaID != "" {
			if media, err := db.GetByID[models.Media](database, cp.MediaID); err == nil {
				viewerTokenURL = media.URL
			}
		}
	}

	embed := commands.CampaignEmbed(*campaign, players, coverURL, viewerTokenURL)
	if campaign.IsArchived {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: messages.CampaignArchivedFooter}
	}

	components := commands.CampaignButtons(userID, *campaign, players, viewerSheetURL)
	components = append(components, helpers.BackRow(router.ViewCampaignsBrowse, "all"))

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
