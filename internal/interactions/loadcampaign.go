package interactions

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
loadDMCampaign authorizes the invoking user as a DM of campaignID and loads
the campaign by ID. On either failure it sends the standard response and
returns (nil, false), so callers can early-return.

The usual flow to get campaigns is:

"authorize -> load -> respond on miss"

This replaces the process with a single method to retrieve Campaigns.

Note:
Handlers that load before authorizing (campaign_leave, campaign_toggle, RenderManageCampaignMenu),
that look up by tag (campaign_join), or that use a different scope
(campaign_approve via checkModAuth) are intentionally omitted.
*/
func loadDMCampaign(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	database *bun.DB,
	campaignID string,
) (*models.Campaign, bool) {
	ok, err := auth.Authorize(database, getUserID(i), auth.ScopeDM, campaignID)
	if err != nil || !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return nil, false
	}
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.ManageCampaignNotFound)
		return nil, false
	}
	return campaign, true
}
