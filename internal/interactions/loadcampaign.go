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

/*
requireMutable rejects archived (or otherwise non-mutable) campaigns by sending
the standard CampaignArchivedMessage response. Returns true when the campaign
is mutable; false when the caller should early-return.

Pair with a load helper at any site that writes to the campaign or fires a
campaign-bound side-effect.
*/
func requireMutable(s *discordgo.Session, i *discordgo.InteractionCreate, c *models.Campaign) bool {
	if !c.CanMutate() {
		respondInteraction(s, i, messages.CampaignArchivedMessage)
		return false
	}
	return true
}

/*
loadModCampaign authorizes the invoking user as a mod (ScopeMod) and loads the
campaign by ID. On either failure it sends the approval-flow response and
returns (nil, false), so callers can early-return.

This is the mod-scoped counterpart to loadDMCampaign, used by the campaign
approval/denial handlers. Sites that only need the auth check (no load) should
keep using checkModAuth.
*/
func loadModCampaign(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	database *bun.DB,
	campaignID string,
) (*models.Campaign, bool) {
	ok, err := auth.Authorize(database, getUserID(i), auth.ScopeMod, "")
	if err != nil || !ok {
		respondInteraction(s, i, messages.CampaignApproveNotModError)
		return nil, false
	}
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignApproveNotFound)
		return nil, false
	}
	return campaign, true
}
