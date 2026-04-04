package interactions

import (
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
)

// AllComponents returns an array with all the `ComponentHandler`s available to the bot.
func AllComponents(db *bun.DB, guildID string, adminRoleName string, d *dispatch.Dispatcher) []ComponentHandler {
	return []ComponentHandler{
		&campaignJoin{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignLeave{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignToggle{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignView{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&manageCampaignMenu{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&manageCampaignDelete{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&manageCampaignBan{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&manageCampaignBanSelect{db: db, guildID: guildID, adminRoleName: adminRoleName},
		&campaignApprove{db: db, dispatcher: d},
		&campaignDeny{db: db},
	}
}

// AllModals returns an array with all the ModalHandlers available to the bot.
func AllModals(db *bun.DB, guildID, adminRole string, d *dispatch.Dispatcher) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db, dispatch: d},
		&campaignDenyModal{db: db, dispatcher: d},
	}
}
