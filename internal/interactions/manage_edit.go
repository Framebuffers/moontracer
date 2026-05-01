package interactions

/*
	Manage Edit handler for the manage campaign menu.

	Flow:
		1. DM clicks "Edit" on the manage campaign menu.
		2. Auth: ScopeDM for that campaign.
		3. Open a modal pre-filled with current campaign values.
		4. On submit, update the campaign in the DB.
		5. Respond with confirmation.

	Discord constraints:
		- Modals support max 5 TextInput fields.
		- No dropdowns or buttons inside modals.
		- Fields to edit (pick 5): Name, Description, Slots, Extra, VTT Link.
		- For edition/format changes, use the existing config dropdowns flow instead.
*/

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type manageEditHandler struct {
	db *bun.DB
}

func (h *manageEditHandler) CustomIDPrefix() string {
	return messages.ManageEditPrefix
}

func (h *manageEditHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	_ = parts[1] // campaignID

	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeDM, campaignID)
	// 2. db.GetByID[models.Campaign](h.db, campaignID)
	// 3. respond with InteractionResponseModal pre-filled with current values:
	//    - Name (TextInputShort, pre-filled)
	//    - Description (TextInputParagraph, pre-filled)
	//    - Max Players (TextInputShort, pre-filled)
	//    - Extra Info (TextInputParagraph, pre-filled)
	//    - VTT Link (TextInputShort, pre-filled)
	respondInteraction(s, i, "Edit is not yet implemented.")
}

type manageEditModal struct {
	db *bun.DB
}

func (h *manageEditModal) CustomIDPrefix() string {
	return messages.ManageEditModalID
}

func (h *manageEditModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopeDM, campaignID)
	// 2. db.GetByID[models.Campaign](h.db, campaignID)
	// 3. parse all 5 fields from modal
	// 4. update campaign fields
	// 5. db.Update(h.db, campaign)
	// 6. respond with confirmation
	respondInteraction(s, i, fmt.Sprintf("Campaign %s updated.", campaignID))
}
