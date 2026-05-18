package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		Triggered when a viewer drills into a single campaign.
		From the browse select menu (campaigns_browse.go), search results, or any nav target
		pointing at ViewCampaignDetail.
		1. RenderCampaignDetail(database, campaignID):
			a. Loads the campaign; replies CampaignNotFound if missing.
			b. Gate: must be approved OR archived. Pending campaigns are hidden
			   from public detail view.
			c. Loads players + cover URL.
			d. If the viewer is a member, loads their CampaignPlayer to surface
			   their assigned token URL and personal sheet URL into the embed/buttons.
			e. Builds embed via commands.CampaignEmbed; if archived, stamps the
			   ArchivedFooter.
			f. Builds context-aware buttons via commands.CampaignButtons (Join/Leave/
			   Manage/Sheet depending on viewer + state).
			g. [Back -> ViewCampaignsBrowse(all)]
			h. Replies with UpdateMessage so it replaces the current ephemeral view.

	Notes:
		- The button set is intentionally computed in the commands package so it
		  stays consistent between the slash-command detail view and this
		  interaction-driven view.
*/

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

	embed := commands.CampaignEmbed(*campaign, players, coverURL, viewerTokenURL, userID)
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
