package interactions

import (
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
)

// AllComponents returns an array with all the `ComponentHandler`s available to the bot.
func AllComponents(db *bun.DB, d *dispatch.Dispatcher) []ComponentHandler {
	return []ComponentHandler{
		&campaignJoin{db: db},
		&campaignLeave{db: db},
		&campaignToggle{db: db},
		&campaignView{db: db},
		&manageCampaignMenu{db: db},
		&manageCampaignDelete{db: db},
		&manageCampaignBan{db: db},
		&manageCampaignBanSelect{db: db},
		&campaignApprove{db: db, dispatcher: d},
		&campaignDeny{db: db},
		&manageCampaignAnnounce{db: db},
		&manageCampaignReschedule{db: db},
	}
}

// AllModals returns an array with all the ModalHandlers available to the bot.
func AllModals(db *bun.DB, d *dispatch.Dispatcher) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db, dispatch: d},
		&campaignDenyModal{db: db, dispatcher: d},
		&manageCampaignAnnounceModal{db: db, dispatcher: d},
		&manageCampaignRescheduleModal{db: db},
	}
}
