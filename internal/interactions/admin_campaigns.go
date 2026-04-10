package interactions

/*
	Admin Campaigns handler for the /admin hub.

	Flow:
		1. Staff clicks "Active Campaigns" on the /admin hub.
		2. Auth: ScopeMod.
		3. Load ALL campaigns (approved + unapproved, not archived).
		4. Render a select menu or paginated list with campaign name, DM, status flags.
		5. Selecting a campaign could show admin-level detail or actions.
		6. Back button returns to /admin hub.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type adminCampaignsHandler struct {
	db *bun.DB
}

func (h *adminCampaignsHandler) CustomIDPrefix() string {
	return messages.AdminCampaignsPrefix
}

func (h *adminCampaignsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeMod, "")
	// 2. db.GetAll[models.Campaign](h.db)
	// 3. filter out archived
	// 4. build select menu or list (max 25 options)
	// 5. back button (messages.BackAdminID)
	respondInteraction(s, i, messages.AdminCampaignsNone)
}
