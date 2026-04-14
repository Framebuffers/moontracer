package messages

/*

	Messages:
		Every single string inside Moontracer is described here.
		This is such that there is a single source of truth for all string values inside the Bot.
		If the dev wants to change any string, they can change it here.

*/

// Generic
const (
	GenericErrorMessage      = "Something went wrong."
	InvalidButtonDataMessage = "Invalid button data."
	BotVersion               = "v1.0-beta"
)

// Command names and descriptions
const (
	PingCommandName        = "ping"
	PingCommandDesc        = "Replies with pong!"
	AwooCommandName        = "awoo"
	AwooCommandDesc        = "do a heccin awoo."
	RegisterCommandName    = "register"
	RegisterCommandDesc    = "Register as a player so you can join and create campaigns."
	CampaignCommandName    = "campaign"
	CampaignCommandDesc    = "Show campaign details."
	MyCampaignsCommandName = "mycampaigns"
	MyCampaignsCommandDesc = "List the campaigns you're part of."
	AddPlayerCommandName   = "add_player"
	AddPlayerCommandDesc   = "Adds a new player to a Campaign."
	TagCommandName         = "tag"
	TagCommandDesc         = "Campaign tag to look up."
)

// Registration
const (
	NotRegisteredMessage       = "You need to `/register` first."
	AlreadyRegisteredMessage   = "You are already registered!"
	RegistrationFailureMessage = "Failed to register. Please try again later."
	RegistrationSuccessMessage = "Welcome, <@%s>! You are now registered."
	RegistrationCheckError     = "register: error checking registration: "
	RegistrationInsertError    = "register: error inserting player: "
)

// Campaign lookup
const (
	CampaignNotFoundMessage         = "Campaign not found."
	CampaignLoadFailureErrorMessage = "Failed to load campaign."
	CampaignPlayersLoadError        = "Failed to load campaign players."
	CampaignFetchError              = "campaign: error fetching campaign %s: "
	PlayerFetchErrorMessage         = "models.GetCampaignPlayers(): Error fetching players: "
)

// Campaign creation
const (
	SlotCountMismatchErrorMessage              = "Invalid slot count. Please enter a number greater than 0."
	CampaignCreationFailureErrorMessage        = "campaign.CreateCampaign(): error creating campaign: "
	CampaignAndRegistrationFailureErrorMessage = "Failed to create campaign. Make sure you are registered."
	CampaignCreationMessage                    = "You just created a new campaign: "
	CampaignStaffNotifyFailureMessage          = "Could not notify staff members to ask for approval of this Campaign."
	CampaignApprovalRequestMessage             = "New campaign **%s** by <@%s> needs approval."
)

// Campaign join
const (
	CampaignClosedMessage          = "This campaign is not open for new players."
	PlayerBannedMessage            = "You are banned from this campaign."
	PlayerAlreadyOnCampaignMessage = "You are already in this campaign."
	CampaignFullMessage            = "This campaign is full."
	PlayerFailedToJoinMessage      = "Failed to join campaign."
	InsertPlayerErrorMessage       = "db.Insert(): Error inserting Campaign Player: "
	PlayerJoinedCampaignMessage    = "You have joined campaign: "
)

// Campaign leave
const (
	MasterIsLeavingCampaignErrorMessage = "You are the DM — you cannot leave your own campaign."
	LeavingCampaignErrorMessage         = "models.RemoveCampaignPlayer(): error removing player: "
	FailedToLeaveCampaignErrorMessage   = "Failed to leave campaign."
	PlayerLeftCampaignMessage           = "You have left campaign "
)

// Campaign toggle
const (
	MasterCanToggleStatusErrorMessage = "Only the DM can toggle campaign status."
	CampaignUpdateErrorMessage        = "db.Update(): error updating campaign: "
	CampaignStatusMessage             = "The current status for campaign"
)

// My campaigns
const (
	NoCampaignsMessage   = "You are not in any campaigns yet."
	MyCampaignsLoadError = "Failed to load your campaigns."
)

// Campaign embed UI labels
const (
	OpenCampaignLabel          = "Set as Open Campaign"
	ClosedCampaignLabel        = "Set as Closed Campaign"
	LeaveCampaignLabel         = "Leave Campaign"
	JoinCampaignLabel          = "Join Campaign"
	EmbedColor                 = 0x5865F2
	ClosedStatusLabel          = "Closed"
	OpenStatusLabel            = "Open"
	CampaignLabel              = "Campaign"
	CampaignTypeOneShotLabel   = "One-shot"
	CampaignTypeWestmarchLabel = "Westmarch"
	NoneLabel                  = "None"
	NoBooksSpecifiedLabel      = "No books specified"
)

// New campaign modal
const (
	NewCampaignModalError    = "newcampaign: error opening modal: "
	NewCampaignModalCustomID = "modal_campaign_create"
	NewCampaignModalTitle    = "Create a New Campaign"
	NewCampaignCommandName   = "newcampaign"
	NewCampaignCommandDesc   = "Create a new campaign (you will be the DM)."
)

// New campaign modal field IDs
const (
	FieldNameID        = "name"
	FieldTagID         = "tag"
	FieldDescriptionID = "description"
	FieldEditionID     = "edition"
	FieldSlotsID       = "slots"
)

// New campaign modal labels
const (
	FieldNameLabel        = "Name"
	FieldTagLabel         = "Tag"
	FieldDescriptionLabel = "Description"
	FieldEditionLabel     = "Edition"
	FieldSlotsLabel       = "Player Slots"
)

// New campaign modal placeholders
const (
	FieldNamePlaceholder        = "e.g. Curse of Strahd"
	FieldTagPlaceholder         = "e.g. curse-of-strahd (short, no spaces)"
	FieldDescriptionPlaceholder = "Describe your campaign setting and premise..."
	FieldEditionPlaceholder     = "e.g. 5e, 3.5e, PF2e"
	FieldSlotsPlaceholder       = "e.g. 4. Set 0 for open campaigns."
)

// Add player
const (
	AddPlayerNotDMOrModMessage   = "You must be the DM of this campaign to add players."
	AddPlayerTargetNotRegistered = "That user is not registered. They need to `/register` first."
	AddPlayerAlreadyInCampaign   = "That player is already in this campaign."
	AddPlayerCampaignFullMessage = "This campaign is full."
	AddPlayerFailureMessage      = "Failed to add player to campaign."
	AddPlayerSuccessMessage      = "Added <@%s> to **%s**!"
)

// Ban
const (
	BanCommandName          = "ban"
	BanCommandDesc          = "Globally ban a player from the server."
	BanCannotBanSelf        = "You cannot ban yourself."
	BanInsufficientRole     = "You cannot ban someone of equal or higher role."
	BanTargetNotFound       = "That player is not registered."
	BanPlayerNotInCampaign  = "The player ID is not valid for this campaign, or it does not belong to it."
	BanTargetAlreadyBanned  = "That player is already banned."
	BanFailureMessage       = "Failed to ban player."
	BanSuccessMessage       = "Banned <@%s>. They can no longer interact with the bot."
	BanSuccessReasonMessage = "Banned <@%s>: %s"
)

// Unban
const (
	UnbanCommandName     = "unban"
	UnbanCommandDesc     = "Lift a global ban from a player."
	UnbanTargetNotFound  = "That player is not registered."
	UnbanTargetNotBanned = "That player is not banned."
	UnbanFailureMessage  = "Failed to unban player."
	UnbanSuccessMessage  = "Unbanned <@%s>. They can interact with the bot again."
)

// Campaign archival
const (
	CampaignArchivedMessage = "This campaign has been archived and can no longer be modified."
	AbandonCommandName      = "abandon"
	AbandonCommandDesc      = "Archive your campaign permanently. Only the DM can do this."
	AbandonNotDMMessage     = "Only the DM of this campaign can abandon it."
	AbandonFailureMessage   = "Failed to archive campaign."
	AbandonSuccessMessage   = "Campaign **%s** has been archived. It is now an immutable record."
	AbandonReasonDM         = "DM abandoned"
	AbandonReasonLeftServer = "DM left server"
)

// Manage campaigns
const (
	ManageCampaignsCommandName = "managecampaigns"
	ManageCampaignsCommandDesc = "Manage campaigns you run as DM."
	ManageNoDMCampaigns        = "You are not the DM of any campaigns."
	ManageNotAuthorized        = "You must be the DM of this campaign to manage it."
	ManageCampaignNotFound     = "Campaign not found."
	ManageDeleteSuccess        = "Campaign **%s** has been deleted."
	ManageDeleteFailure        = "Failed to delete campaign."
	ManageBanNoMembers         = "This campaign has no members to ban."
	ManageCampaignBanSuccess   = "Banned <@%s> from **%s**."
	ManageCampaignBanFailure   = "Failed to ban player from campaign."
	ManageSelectMember         = "Select a member to ban from **%s**:"
)

// Manage campaigns — button labels
const (
	ManageEditLabel       = "Edit"
	ManageDeleteLabel     = "Delete"
	ManageBanLabel        = "Ban Member"
	ManageAnnounceLabel   = "Announce"
	ManageRescheduleLabel      = "Reschedule"
	ManageCampaignButtonLabel  = "Manage"
)

// Set campaign role
const (
	SetCampaignRoleCommandName = "setcampaignrole"
	SetCampaignRoleCommandDesc = "Link a Discord role to a campaign (creates one if it doesn't exist)."
	SetRoleFieldName           = "role"
	SetRoleFieldDesc           = "Name of the Discord role to link."
	SetRoleNotDMOrMod          = "You must be the DM of this campaign to set its role."
	SetRoleSuccess             = "Linked role **%s** to campaign **%s**."
	SetRoleCreateFailed        = "Failed to create Discord role."
	SetRoleUpdateFailed        = "Failed to update campaign."
)

// Campaign approval
const (
	CampaignApprovePrefix         = "campaign_approve"
	CampaignDenyPrefix            = "campaign_deny"
	CampaignDenyModalPrefix       = "campaign_deny_modal"
	ApproveButtonLabel            = "Approve"
	DenyButtonLabel               = "Deny"
	CampaignApprovedMessage       = "Campaign **%s** has been approved."
	CampaignDeniedMessage         = "Campaign **%s** has been denied and deleted."
	CampaignApproveNotModError    = "Only mods or admins can approve campaigns."
	CampaignApproveNotFound       = "Campaign not found or already processed."
	CampaignApproveError          = "Failed to process campaign approval."
	CampaignDenyModalTitle        = "Deny Campaign"
	CampaignDenyReasonLabel       = "Reason"
	CampaignDenyReasonPlaceholder = "Why is this campaign being denied?"
	CampaignDenyReasonFieldID     = "deny_reason"
	CampaignDeniedDMMessage       = "Your campaign **%s** has been denied. Reason: %s"
	CampaignApprovedDMMessage     = "Your campaign **%s** has been approved!"
	CampaignApprovedStatusMessage = "Approved campaign **%s**."
	CampaignDeniedStatusMessage   = "Denied campaign **%s**. Reason: %s"
	CampaignDenyPendingMessage    = "Campaign **%s** — denial in progress..."
)

// Announce
const (
	AnnounceModalPrefix      = "manage_announce_modal"
	AnnounceComponentPrefix  = "manage_announce"
	AnnounceModalTitle       = "Send Announcement"
	AnnounceFieldID          = "announce_message"
	AnnounceFieldLabel       = "Message"
	AnnounceFieldPlaceholder = "Type your announcement to all campaign members..."
	AnnounceSentMessage      = "Announcement sent to %d members of **%s**."
	AnnounceNoMembers        = "This campaign has no members to announce to."
	AnnounceError            = "Failed to send announcement."
	AnnouncePostedToThread   = "Announcement posted to the **%s** announcements thread."
)

// Reschedule
const (
	RescheduleModalPrefix      = "manage_reschedule_modal"
	RescheduleComponentPrefix  = "manage_reschedule"
	RescheduleModalTitle       = "Reschedule Campaign"
	RescheduleDayFieldID       = "reschedule_day"
	RescheduleDayLabel         = "Day of Week (0=Mon, 1=Tue, 2=Wed, 3=Thu, 4=Fri, 5=Sat, 6=Sun)"
	RescheduleDayPlaceholder   = "e.g. 5 for Saturday"
	RescheduleTimeFieldID      = "reschedule_time"
	RescheduleTimeLabel        = "Start Time (HH:MM UTC)"
	RescheduleTimePlaceholder  = "e.g. 19:00"
	RescheduleDurFieldID       = "reschedule_duration"
	RescheduleDurLabel         = "Duration (hours)"
	RescheduleDurPlaceholder   = "e.g. 3"
	RescheduleFreqFieldID      = "reschedule_freq"
	RescheduleFreqLabel        = "Frequency (weekly, biweekly, monthly, quarterly, yearly)"
	RescheduleFreqPlaceholder  = "e.g. weekly"
	RescheduleSuccess          = "Schedule updated for **%s**: %s %s UTC (%sh), %s."
	RescheduleInvalidDay       = "Invalid day of week. Use 0 (Mon) through 6 (Sun)."
	RescheduleInvalidTime      = "Invalid time format. Use HH:MM (e.g. 19:00)."
	RescheduleInvalidDuration  = "Invalid duration. Enter a number (e.g. 3)."
	RescheduleInvalidFrequency = "Invalid frequency. Use: weekly, biweekly, monthly, quarterly, yearly."
	RescheduleError            = "Failed to update schedule."
)

// Campaign database (debug)
const (
	CampaignDBCommandName = "campaigndatabase"
	CampaignDBCommandDesc = "Show all campaigns in the database (staff only)."
	CampaignDBEmpty       = "No campaigns in the database."
	CampaignDBNotStaff    = "Only mods or admins can use this command."
)

// Help command
const (
	HelpCommandName = "help"
	HelpCommandDesc = "Get a list of all available commands."
)

// Player hub (/me)
const (
	MeCommandName = "me"
	MeCommandDesc = "Your player profile and quick actions."
	MeHubMessage  = "Hey, <@%s>! What would you like to do?"
)

// Browse campaigns (/campaigns)
const (
	CampaignsCommandName       = "campaigns"
	CampaignsCommandDesc       = "Browse all available campaigns."
	CampaignsFilterPlaceholder = "Filter by format..."
	CampaignsSelectPlaceholder = "Select a campaign..."
	CampaignsNoneAvailable     = "No campaigns match this filter."
	CampaignsFilterAll         = "All"
	CampaignsFilterCampaign    = "Campaigns"
	CampaignsFilterOneshot     = "One-shots"
	CampaignsFilterWestmarch   = "Westmarches"
)

// Search (/search)
const (
	SearchCommandName = "search"
	SearchCommandDesc = "Search for a campaign by name."
	SearchOptionName  = "name"
	SearchOptionDesc  = "Campaign name to search for."
	SearchNoResults   = "No campaigns found matching that name."
)

// Admin hub (/admin)
const (
	AdminCommandName    = "admin"
	AdminCommandDesc    = "Mod/Admin panel."
	AdminNotStaff       = "Only mods or admins can use this command."
	AdminHubMessage     = "Admin Panel:"
	AdminCampaignsLabel = "Active Campaigns"
	AdminDMsLabel       = "DMs"
	AdminBroadcastLabel = "Broadcast"
	AdminDatabaseLabel  = "Database"
	AdminSettingsLabel  = "Settings"
	AdminDiagLabel      = "Diagnostics"
)

// About (/moontracer)
const (
	AboutCommandName           = "moontracer"
	AboutCommandDesc           = "About this bot."
	AboutCommandGitHubRepoLink = "https://github.com/framebuffers/moontracer"
	AboutCommandWebsite        = "https://framebuffer.cl/moontracer"
	AboutCommandBotDesc        = "Moontracer: a D&D campaign manager for players, DM and spectators!"
	AboutCommandCopyright      = "(C) 2026 Framebuffer"
	AboutCommandLicense        = "AGPL-v3.0"
	AboutCommandHelp           = "Type `/help` for a list of commands."
	AboutCommandAwoo           = "awoo!"
	AboutCommandAttributions   = "Thanks to the D&D r/Chile Discord server for giving me the idea, letting me test the bot on their server and give me feedback to improve this bot."
)

// Navigation buttons
const (
	BackLabel         = "Back"
	BackMeID          = "back_me"
	BackCampaignsID   = "back_campaigns"
	BackMyCampaignsID = "back_mycampaigns"
	BackManageID      = "back_manage"
	BackAdminID       = "back_admin"
)

// Hub button labels
const (
	MyCampaignsLabel     = "My Campaigns"
	NextSessionsLabel    = "Next Sessions"
	NotificationsLabel   = "Notifications"
	BrowseCampaignsLabel = "Browse Campaigns"
	MyProfileLabel       = "My Profile"
	NewCampaignLabel     = "New Campaign"
)

// Manage campaign: additional buttons
const (
	ManageSetRoleLabel  = "Set Role"
	ManageArchiveLabel  = "Archive"
	ManageSetRolePrefix = "manage_role"
	ManageArchivePrefix = "manage_archive"

	// Set Role modal
	ManageSetRoleModalTitle = "Link Discord Role"
	ManageSetRoleFieldLabel = "Role name (creates if it doesn't exist)"
	ManageSetRoleModalID    = "modal_manage_role"
	ManageSetRoleSuccess    = "Linked role **%s** to campaign **%s**."
	ManageSetRoleFailed     = "Failed to set role."

	// Delete confirmation + handler
	ManageDeleteConfirm      = "Are you sure you want to delete **%s**? This is permanent and cannot be undone. All members will be removed."
	ManageDeleteConfirmID    = "manage_delete_confirm"
	ManageDeleteConfirmLabel = "Yes, Delete"
	ManageDeleteCancelLabel  = "Cancel"

	// Archive confirmation + handler
	ManageArchiveConfirm      = "Are you sure you want to archive **%s**? This is permanent and cannot be undone."
	ManageArchiveConfirmID    = "manage_archive_confirm"
	ManageArchiveCancelID     = "manage_archive_cancel"
	ManageArchiveConfirmLabel = "Yes, Archive"
	ManageArchiveCancelLabel  = "Cancel"
	ManageArchiveSuccess      = "Campaign **%s** has been archived. It is now an immutable record."
	ManageArchiveFailed       = "Failed to archive campaign."
)

// New campaign config (post-modal dropdowns)
const (
	NewCampaignBookPrefix        = "newcampaign_book"
	NewCampaignFormatPrefix      = "newcampaign_format"
	NewCampaignSubmitPrefix      = "newcampaign_submit"
	NewCampaignCancelPrefix      = "newcampaign_cancel"
	NewCampaignBookPlaceholder   = "Select a game system..."
	NewCampaignFormatPlaceholder = "Select a format..."
	NewCampaignConfigMessage     = "Campaign **%s** created (pending setup).\n\nSelect a game system and format, then submit for approval."
	NewCampaignSubmitLabel       = "Submit for Approval"
	NewCampaignCancelLabel       = "Cancel"
	NewCampaignSubmittedMessage  = "Campaign **%s** has been submitted for approval!"
	NewCampaignCancelledMessage  = "Campaign creation cancelled."

	// Headers shown above the configuration dropdowns
	NewCampaignConfigHeader       = "Configure **%s**:"
	NewCampaignConfigSystemHeader = "Configure **%s** — system: **%s**"
	NewCampaignConfigFormatHeader = "Configure **%s** — format: **%s**"

	// Book dropdown option labels
	NewCampaignBookLabel5e    = "D&D 5e"
	NewCampaignBookLabel55e   = "D&D 5.5e (2024)"
	NewCampaignBookLabelPF2e  = "Pathfinder 2e"
	NewCampaignBookLabelOther = "Other / Homebrew"
)

// Campaign Modal
const (
	FieldSlotsPlaceholderNew = "e.g. 4 (leave empty for unlimited)"
	FieldSynopsisLabel       = "Synopsis & Rules"
)

// Select menu placeholders + content prefixes for /mycampaigns and /managecampaigns
const (
	MyCampaignsPlaceholder     = "Select a campaign..."
	ManageCampaignsPlaceholder = "Select a campaign to manage..."
	MyCampaignsListHeader      = "Your campaigns:\n"
	ManageCampaignsListHeader  = "Your campaigns (DM):\n"
)

// Select menu CustomIDs
const (
	CampaignSelectPrefix      = "campaign_select"
	MyCampaignSelectPrefix    = "mycampaign_select"
	ManageSelectPrefix        = "manage_select"
	CampaignsFilterPrefix     = "campaigns_filter"
	AdminCampaignSelectPrefix = "admin_campaign_select"
)

// Player hub: Next Sessions
const (
	NextSessionsPrefix  = "next_sessions"
	NextSessionsHeader  = "Upcoming sessions:"
	NextSessionsNone    = "You have no upcoming sessions."
)

// Player hub: Notifications
const (
	NotificationsPrefix  = "notifications"
	NotificationsHeader  = "Notification settings:"
	NotificationsNone    = "No notification preferences configured yet."
)

// Admin hub: Campaign browser (all campaigns)
const (
	AdminCampaignsPrefix     = "admin_campaigns"
	AdminCampaignsHeader     = "All campaigns:"
	AdminCampaignsNone       = "No campaigns in the database."
)

// Admin hub: Broadcast
const (
	AdminBroadcastPrefix       = "admin_broadcast"
	AdminBroadcastModalID      = "modal_admin_broadcast"
	AdminBroadcastModalTitle   = "Broadcast Message"
	AdminBroadcastFieldLabel   = "Message"
	AdminBroadcastSuccess      = "Broadcast sent."
	AdminBroadcastFailed       = "Failed to send broadcast."
)

// Admin hub: Database viewer
const (
	AdminDatabasePrefix = "admin_database"
)

// Admin hub: Settings
const (
	AdminSettingsPrefix = "admin_settings"
	AdminSettingsHeader = "Bot settings:"
)

// Admin hub: Diagnostics
const (
	AdminDiagPrefix = "admin_diag"
)

// Manage campaign: Edit
const (
	ManageEditPrefix     = "manage_edit"
	ManageEditModalID    = "modal_manage_edit"
	ManageEditModalTitle = "Edit Campaign"
)

// Manage campaign: New Campaign from button (modal-from-component)
const (
	ManageNewCampaignPrefix = "manage_newcampaign"
)

// Manage campaign: Invite Player
const (
	ManageInviteLabel        = "Invite Player"
	ManageInvitePrefix       = "manage_invite"
	ManageInviteSelectPrefix = "manage_invite_select"
	InviteAcceptPrefix       = "campaign_invite_accept"
	InviteDeclinePrefix      = "campaign_invite_decline"
	InviteSentMessage        = "Invitation sent to <@%s> for **%s**."
	InviteDMMessage          = "You've been invited to join **%s** by <@%s>!"
	InviteAcceptedDMUpdate   = "You accepted the invitation to **%s**."
	InviteDeclinedDMUpdate   = "You declined the invitation to **%s**."
	InviteAlreadyProcessed   = "This invitation has already been processed."
	InviteCampaignFull       = "Cannot invite — campaign **%s** is full."
)
