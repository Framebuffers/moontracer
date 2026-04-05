package interactions

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

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
