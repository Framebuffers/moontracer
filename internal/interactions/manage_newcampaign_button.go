package interactions

/*
	New Campaign from button handler for the /managecampaigns menu.

	Flow:
		1. DM clicks "New Campaign" button on /managecampaigns.
		2. Respond with InteractionResponseModal (same modal as /newcampaign slash command).
		3. Modal submission is handled by the existing modalCampaignCreate handler.

	The key challenge:
		The existing modal-create flow (modalCampaignCreate) was built around a slash-command
		entry point. Opening the same modal from a component button interaction requires that
		the handler router supports InteractionResponseModal from component interactions.

		The router in handler.go dispatches component interactions to ComponentHandler.HandleComponents,
		which can respond with any InteractionResponse type — including InteractionResponseModal.
		So the wiring works, we just need to build the same modal response here.

	Note:
		The modal's CustomID must match what modalCampaignCreate expects so the existing
		submit handler picks it up.
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/messages"
)

type manageNewCampaignButton struct {
	db *bun.DB
}

func (h *manageNewCampaignButton) CustomIDPrefix() string {
	return messages.ManageNewCampaignPrefix
}

func (h *manageNewCampaignButton) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: implement
	// 1. auth.Authorize(h.db, getUserID(i), auth.ScopePlayer, "")
	// 2. respond with InteractionResponseModal using the same modal definition
	//    as newCampaign.Execute in commands/new_campaign.go
	//    CustomID must be messages.NewCampaignModalID (or whatever modalCampaignCreate expects)
	//    so the existing modal submit handler picks it up seamlessly
	//
	// Note: check what CustomID modalCampaignCreate uses for its prefix,
	// and build the identical modal here.
	respondInteraction(s, i, "New Campaign from button is not yet implemented.")
}
