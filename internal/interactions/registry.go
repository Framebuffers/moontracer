package interactions

import (
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/scheduler"
)

/*
AllComponents returns an array with all the `ComponentHandler`s available to the bot.

Also registers all router views. It's called once at startup.

How to add a new component (button / select menu) interaction:

 1. Create internal/interactions/<name>.go with a struct that holds
    its dependencies (usually `db *bun.DB`, sometimes the dispatcher).

 2. Implement the ComponentHandler interface (interaction.go):

    - CustomIDPrefix() string. It's the "<prefix>" part of CustomIDs of
    the form "prefix:arg1:arg2". Must be unique across all handlers.

    - HandleComponents(s, i). It's invoked when a button/select whose
    CustomID starts with this prefix fires.

 3. Emit the CustomID from wherever the button is rendered, as
    fmt.Sprintf("%s:%s", prefix, someID). Keep the prefix literal in
    internal/messages/messages.go if it's shared, or as a constant on
    the handler file if it's local.

 4. Add the handler to the slice below, grouped with its neighbors
    (Campaign actions, Manage, Approval, Browse, Admin, etc).

 5. User-facing strings go in internal/messages/messages.go.

 6. If the handler needs auth, call auth.Authorize at the top. Same
    scopes as commands.“

How to add a new modal:

 1. Implement the ModalHandler interface (CustomIDPrefix + HandleModal).

 2. Register it in AllModals() below.

 3. Trigger it by responding to a prior interaction with
    InteractionResponseModal and a matching CustomID.

How to add a new router view (universal back-button support):

	Views are addressable screens (ViewMe, ViewManage, ViewAdmin...) that
	render via navHandler. Add new views in internal/interactions/views.go
	by calling router.Register(router.ViewDragon, renderFn). The back button
	then navigates by ViewID, so any new screen that wants Back support
	needs to be a registered view.

At startup the bot calls AllComponents/AllModals once per guild DB; the
main handler (internal/discord/handler.go) dispatches incoming events to
the first handler whose prefix matches.
*/
func AllComponents(db *bun.DB, d *dispatch.Dispatcher, sched *scheduler.Scheduler, dataDir, mediaBaseURL string) []ComponentHandler {
	RegisterAllViews(db, d)

	return []ComponentHandler{
		// Campaign actions
		&campaignJoin{db: db, dispatcher: d},
		&campaignLeave{db: db},
		&campaignToggle{db: db},
		&campaignView{db: db},

		// Campaign management
		&manageCampaignMenu{db: db},
		&manageCampaignDelete{db: db},
		&manageDeleteConfirm{db: db},
		&manageCampaignBan{db: db},
		&manageCampaignBanSelect{db: db},
		&manageCampaignAnnounce{db: db},
		&manageCampaignReschedule{db: db},
		&manageSetRole{db: db},
		&manageArchive{db: db},
		&manageArchiveConfirm{db: db, sched: sched},
		&manageSetCover{db: db},

		// Approval (DM flow)
		&campaignApprove{db: db, dispatcher: d},
		&campaignDeny{db: db},

		// Browse & select
		&campaignsFilterHandler{db: db},
		&campaignSelectHandler{db: db},
		&myCampaignSelectHandler{db: db},
		&manageSelectHandler{db: db},

		// Navigation (all "nav:*" CustomIDs go through the view router).
		&navHandler{db: db},

		// Quick registration button (shown on all "not registered" surfaces).
		&quickRegisterHandler{db: db},

		// Session announce + response
		&manageNewSessionButton{db: db},
		&sessionResponseAcceptHandler{db: db, dispatcher: d},
		&sessionResponseDeclineHandler{db: db, dispatcher: d},
		&sessionResponseConfirmHandler{db: db, dispatcher: d},
		&sessionResponseCancelHandler{},
		&sessionResponseRetractHandler{db: db, dispatcher: d},
		&sessionConflictHandler{db: db, dispatcher: d},
		&sessionConflictSelHandler{db: db, dispatcher: d},

		// New campaign config (post-modal)
		&newCampaignBookHandler{db: db},
		&newCampaignFormatHandler{db: db},
		&newCampaignFrequencyHandler{db: db},
		&newCampaignSubmitHandler{db: db, dispatcher: d},
		&newCampaignCancelHandler{db: db},

		// Player hub
		&nextSessionsHandler{db: db},
		&notificationsHandler{db: db},
		&notifToggleHandler{db: db},
		&timezoneButtonHandler{db: db},
		&timezoneSelectHandler{db: db},

		// Admin hub
		&adminCampaignsHandler{db: db},
		&adminCampaignSelectHandler{db: db},
		&adminBroadcastHandler{db: db, dispatcher: d},
		&adminDatabaseHandler{db: db},
		&adminSettingsHandler{db: db},
		&adminBillboardSetHandler{db: db},
		&adminBillboardSetCategoryHandler{db: db},
		&adminCampaignsCategoryHandler{db: db},
		&adminArchivedCategoryHandler{db: db},
		&adminCampaignChannelSetHandler{db: db},
		&adminDiagHandler{db: db},

		// Manage: new campaign from button, links, player tokens
		&manageNewCampaignButton{db: db},
		&manageLinksHandler{db: db},
		&manageDownloadTokensHandler{db: db},

		// Token generator confirm/discard/assign
		&tokenApplyHandler{db: db, dataDir: dataDir, mediaBaseURL: mediaBaseURL},
		&tokenDiscardHandler{dataDir: dataDir, mediaBaseURL: mediaBaseURL},
		&playerTokenPostcreateSelectHandler{db: db},
		&playerTokenSkipHandler{db: db},
		&manageSetSession{db: db},

		// Token gallery (/me -> Tokens)
		&tokenGallerySelectHandler{db: db},
		&tokenGalleryAssignHandler{db: db},
		&tokenGalleryAssignSelectHandler{db: db},
		&tokenDeletePromptHandler{db: db},
		&tokenDeleteConfirmHandler{db: db, dataDir: dataDir},
		&tokenDownloadHandler{db: db},

		// Player campaign card menus
		&playerSetSheetHandler{db: db},
		&playerSetTokenHandler{db: db},
		&playerTokenSelectHandler{db: db},
		&playerTokenAssignHandler{db: db},
		&playerLeaveConfirmPromptHandler{db: db},
		&playerLeaveDoHandler{db: db},
		&playerContactDMHandler{db: db},
		&playerDownloadTokensHandler{db: db},
		&playerDownloadSelectHandler{db: db},

		// Session response (buttons on reminder DMs)
		&responseAcceptHandler{db: db, dispatcher: d},
		&responseDeclineHandler{db: db, dispatcher: d},

		// Invitations
		&manageCampaignInvite{db: db, dispatcher: d},
		&manageCampaignInviteSelect{db: db, dispatcher: d},
		&campaignInviteAccept{db: db},
		&campaignInviteDecline{db: db},

		// Campaign import thread-mapping flow
		&importThreadSelHandler{},
		&importNextHandler{},
		&importBackHandler{},
		&importCancelHandler{},
		&importConfirmHandler{db: db},

		// Campaign import billboard channel selector
		&importBillboardSelHandler{db: db},
		&importBillboardSkipHandler{db: db},
	}
}

// AllModals returns an array with all the ModalHandlers available to the bot.
func AllModals(db *bun.DB, d *dispatch.Dispatcher, sched *scheduler.Scheduler, dataDir, mediaBaseURL string) []ModalHandler {
	return []ModalHandler{
		&modalCampaignCreate{db: db, dispatch: d},
		&newCampaignScheduleModal{db: db, dispatcher: d},
		&newSessionModal{db: db, dispatcher: d, sched: sched},
		&campaignDenyModal{db: db, dispatcher: d},
		&manageCampaignAnnounceModal{db: db, dispatcher: d},
		&manageCampaignRescheduleModal{db: db},
		&manageSetRoleModal{db: db},
		&manageLinksModal{db: db},
		&manageSetSessionModal{db: db, sched: sched},
		&adminBroadcastModal{db: db, dispatcher: d},
		&playerSetSheetModal{db: db},
		&playerContactDMModal{db: db, dispatcher: d},
		&tokenApplyModal{db: db, dataDir: dataDir, mediaBaseURL: mediaBaseURL},
	}
}
