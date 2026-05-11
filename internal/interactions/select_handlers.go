package interactions

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

/*
	Flow:
		Triggered when a player picks an entry from one of the two list-style
		select menus rendered in back_handlers.go. Each handler is a thin
		forwarder to the appropriate render function:

		1. mycampaign_select: myCampaignSelectHandler
			a. Selection -> RenderCampaignDetail (campaign_detail.go).
			b. Sentinel "none" (empty-list placeholder) is silently ignored.
		2. manage_select: manageSelectHandler
			a. Selection -> RenderManageCampaignMenu (manage_campaign.go).
			b. Same "none" sentinel handling.

	Notes:
		- "none" comes from BuildPlayerCampaignSelect's empty-state fallback in
		  components.go; both menus reuse that builder.
		- Auth is not enforced here; the render functions both gate themselves
		  (RenderManageCampaignMenu via DM-scope auth, RenderCampaignDetail by
		  approval state).
*/

// myCampaignSelectHandler handles mycampaign_select: player picked a campaign from /mycampaigns.
type myCampaignSelectHandler struct {
	db *bun.DB
}

func (h *myCampaignSelectHandler) CustomIDPrefix() string {
	return messages.MyCampaignSelectPrefix
}

func (h *myCampaignSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	if len(values) == 0 || values[0] == "none" {
		return
	}
	RenderCampaignDetail(s, i, h.db, values[0])
}

// manageSelectHandler handles manage_select: DM picked a campaign from /managecampaigns.
type manageSelectHandler struct {
	db *bun.DB
}

func (h *manageSelectHandler) CustomIDPrefix() string {
	return messages.ManageSelectPrefix
}

func (h *manageSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	if len(values) == 0 || values[0] == "none" {
		return
	}
	RenderManageCampaignMenu(s, i, h.db, values[0])
}
