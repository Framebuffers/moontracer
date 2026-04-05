package interactions

import (
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
)

// AllComponents returns an array with all the `ComponentHandler`s available to the bot.
func AllComponents(db *bun.DB, d *dispatch.Dispatcher) []ComponentHandler {
	return []ComponentHandler{
		// Campaign actions
		&campaignJoin{db: db},
		&campaignLeave{db: db},
		&campaignToggle{db: db},
		&campaignView{db: db},

		// Campaign management
		&manageCampaignMenu{db: db},
		&manageCampaignDelete{db: db},
		&manageCampaignBan{db: db},
		&manageCampaignBanSelect{db: db},
		&manageCampaignAnnounce{db: db},
		&manageCampaignReschedule{db: db},

		// Approval (DM flow)
		&campaignApprove{db: db, dispatcher: d},
		&campaignDeny{db: db},

		// Browse & select
		&campaignsFilterHandler{db: db},
		&campaignSelectHandler{db: db},
		&myCampaignSelectHandler{db: db},
		&manageSelectHandler{db: db},

		// Back navigation
		&backMe{db: db},
		&backMyCampaigns{db: db},
		&backManage{db: db},
		&backCampaigns{db: db},
		&backManageCampaign{db: db},

		// Stubs
		&stubHandler{},
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
