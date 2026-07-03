package interactions

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
manageRefreshBillboard re-renders the billboard starter message and, if the campaign is open,
posts a fresh join button to the thread.

Note: any existing join button message is not deleted first — the DM cleans up stale ones manually.

CustomID: manage_refresh_billboard:<campaignID>
*/
type manageRefreshBillboard struct {
	db *bun.DB
}

func (h *manageRefreshBillboard) CustomIDPrefix() string {
	return messages.ManageRefreshBillboardPrefix
}

func (h *manageRefreshBillboard) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := renderManageSubAuth(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if err := helpers.UpdateBillboard(s, h.db, campaign); err != nil {
		log.Printf("manage_refresh_billboard: update billboard for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageRefreshBillboardFailed)
		return
	}

	if campaign.IsOpen && campaign.BillboardThreadID != "" {
		postJoinButton(s, campaign.BillboardThreadID, campaign)
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageRefreshBillboardSuccess, campaign.Name))
}
