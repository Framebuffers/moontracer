package interactions

import "github.com/bwmarrin/discordgo"

/*
	Note:
		These handlers, when going to the dispatcher (handler.go), go through this switch statement that decides which handler to execute:
			```go
				switch i.Type {
				case discordgo.InteractionApplicationCommand:
					cmd.Execute(s, i)
				case discordgo.InteractionMessageComponent: // ComponentHandler
					handler.HandleComponents(s, i)
				case discordgo.InteractionModalSubmit: 		// ModalHandler
					handler.HandleModals(s, i)
				}
			```
*/

// ComponenHandler defines a single routing string, plus its handler, that all Components must implement.
// It handles button clicks and interactions like selecting menus, etc.
// Flow: campaignJoin handles the "Join Campaign" button -> HandleComponents(s, i) -> gets registered handlers `[campaignJoin, campaignLeave, campaignToggel, campaignView]`
type ComponentHandler interface {

	// CustomIDPrefix tells Moontracer which handler to call, and the params it expects to perform its tasks.
	// convention = prefix:arg (e.g. `join_campaign:strahd`)
	CustomIDPrefix() string
	HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// ModalHandler defines a single identification string, plus its handler, that all Modals must implement, in specific, modal form submissions/
// This handles the user interface in charge of user text input.
// Flow: modalCampaignCreate handles the /newcampaign modal form submission -> HandleModal(s, i) -> InteractionModalSubmit routed to ModalHandler.
type ModalHandler interface {
	CustomIDPrefix() string
	HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate)
}
