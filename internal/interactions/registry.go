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
		&manageSetRole{db: db},
		&manageArchive{db: db},
		&manageArchiveConfirm{db: db},

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
		&backAdmin{db: db},
		&backManageCampaign{db: db},

		// New campaign config (post-modal)
		&newCampaignBookHandler{db: db},
		&newCampaignFormatHandler{db: db},
		&newCampaignSubmitHandler{db: db, dispatcher: d},
		&newCampaignCancelHandler{db: db},

		// Player hub
		&nextSessionsHandler{db: db},
		&notificationsHandler{db: db},

		// Admin hub
		&adminCampaignsHandler{db: db},
		&adminBroadcastHandler{db: db, dispatcher: d},
		&adminDatabaseHandler{db: db},
		&adminSettingsHandler{db: db},
		&adminDiagHandler{db: db},

		// Manage: edit + new campaign from button
		&manageEditHandler{db: db},
		&manageNewCampaignButton{db: db},
	}
}

// AllModals returns an array with all the ModalHandlers available to the bot.
func AllModals(db *bun.DB, d *dispatch.Dispatcher) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db, dispatch: d},
		&campaignDenyModal{db: db, dispatcher: d},
		&manageCampaignAnnounceModal{db: db, dispatcher: d},
		&manageCampaignRescheduleModal{db: db},
		&manageSetRoleModal{db: db},
		&manageEditModal{db: db},
		&adminBroadcastModal{db: db, dispatcher: d},
	}
}
