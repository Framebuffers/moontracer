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
	BotVersion               = "v0.10.2"
)

/*
Date/time formats.

There are two different kinds of formats: input and display.

Input formats are interface contracts: they describe what DMs type into
modals and aren't locale-dependent.

Display formats render human-readable strings and could be localized:
to translate them, swap these per locale to get day/month names
in the user's language.

NOTE:

	Go's time package uses the reference time `Mon Jan 2 15:04:05 MST 2006` to
	define layouts; see https://pkg.go.dev/time#Layout.
*/
const (
	// Input: what the DM types (and it's parsed).
	DateInputFormat     = "2006-01-02"       // YYYY-MM-DD (ISO 8601 date)
	TimeInputFormat     = "15:04"            // HH:MM (24h)
	DateTimeInputFormat = "2006-01-02 15:04" // combined for ParseInLocation

	// Display: these can be translated later when localization can occur.
	SessionTimeFormat = "Mon 2 Jan 2006 15:04" // long form, includes year
	SessionListFormat = "Mon 2 Jan 15:04"      // compact, year implied
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
	CampaignArchivedFooter          = "This campaign is archived — it is an immutable record."
	CampaignLoadFailureErrorMessage = "Failed to load campaign."
	CampaignPlayersLoadError        = "Failed to load campaign players."
	CampaignFetchError              = "campaign: error fetching campaign %s: "
	PlayerFetchErrorMessage         = "models.GetCampaignPlayers(): Error fetching players: "
)

// Campaign creation
const (
	SlotCountMismatchErrorMessage              = "Invalid slot count. Capacity must be a positive number. Leave the field empty for unlimited."
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
	PlayerJoinedCampaignMessage    = "You have joined **%s**!"

	// Westmarch session-capacity tripwire (FCFS soft alert).
	WestmarchOverCapacityDMAlert      = "⚠️ **INFO:** <@%s> just joined westmarch **%s**. Roster is now %d active player(s). Session capacity is %d. They've been admitted; bring an extra seat or trim attendance for the next session."
	WestmarchOverCapacityPlayerNotice = "You're in **%s**! Warning!: this westmarch's session capacity (%d) is already met, so the DM has been notified. Talk to your DM for more help."
)

// Campaign leave
const (
	MasterIsLeavingCampaignErrorMessage = "You are the DM — you cannot leave your own campaign."
	LeavingCampaignErrorMessage         = "models.RemoveCampaignPlayer(): error removing player: "
	FailedToLeaveCampaignErrorMessage   = "Failed to leave campaign."
	PlayerLeftCampaignMessage           = "You have left **%s**."
)

// Campaign toggle
const (
	MasterCanToggleStatusErrorMessage = "Only the DM can toggle campaign status."
	CampaignUpdateErrorMessage        = "db.Update(): error updating campaign: "
	CampaignStatusMessage             = "**%s** is now **%s**."
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
	ArchivedStatusLabel        = "Archived"
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
	ManageCampaignsLabel       = "Manage Campaigns"
	ManageNoDMCampaigns        = "You are not the DM of any campaigns."
	ManageNotAuthorized        = "You must be the DM of this campaign to manage it."
	ManageCampaignNotFound     = "Campaign not found."
	ManageDeleteSuccess        = "Campaign **%s** has been deleted."
	ManageDeleteFailure        = "Failed to delete campaign."
	ManageBanNoMembers         = "This campaign has no members to ban."
	ManageCampaignBanSuccess   = "Banned <@%s> from **%s**."
	ManageCampaignBanFailure   = "Failed to ban player from campaign."
	ManageSelectMember         = "Select a member to ban from **%s**:"
	ManageBanSelectPlaceholder = "Select a member..."
	ManageCampaignHeader       = "Managing **%s**:"
	ManageInviteSelectPrompt   = "Select a player to invite to **%s**:"
	MyCampaignListLine         = "**%s** — %s (%s)"
	ManageCampaignListLine     = "**%s** — %s"
)

// Campaign cover / upload
const (
	CampaignUploadCommandName     = "campaignupload"
	CampaignUploadCommandDesc     = "Upload an image for one of your campaigns."
	CampaignUploadKindOptName     = "kind"
	CampaignUploadKindOptDesc     = "What kind of image to upload."
	CampaignUploadKindCoverChoice = "Cover"
	CampaignUploadCampaignOptName = "campaign"
	CampaignUploadCampaignOptDesc = "The campaign to upload an image for."
	CampaignUploadImageOptName    = "image"
	CampaignUploadImageOptDesc    = "Image file (JPEG/PNG/WebP, up to 8 MiB)."

	CampaignUploadNotDM         = "Only the DM of this campaign can change its cover."
	CampaignUploadNotImage      = "That file doesn't look like an image. Try JPEG, PNG, or WebP."
	CampaignUploadTooLarge      = "Image is too large. Max 8 MiB."
	CampaignUploadMissingAttach = "No image attached. Attach a file to the `image` option."
	CampaignUploadFailure       = "Failed to save cover. Please try again."
	CampaignUploadSuccess       = "Cover set for **%s**. [View](%s)"

	SetCoverButtonLabel  = "Set Cover"
	SetCoverInstructions = "Use `/campaignupload kind:Cover campaign:<name> image:<file>` to set a cover for this campaign."
)

// Manage campaigns — button labels
const (
	ManageDeleteLabel         = "Delete"
	ManageBanLabel            = "Ban Member"
	ManageAnnounceLabel       = "Announce"
	ManageRescheduleLabel     = "Configure Schedule"
	ManageCampaignButtonLabel = "Manage"
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
	AnnounceThreadContent    = "%s**Announcement from <@%s>:**\n\n%s"
	AnnounceDMContent        = "**[%s]** Announcement from <@%s>:\n\n%s"
)

// CampaignsCategoryName is the shared Discord category all campaign channels are grouped under.
const CampaignsCategoryName = "Campaigns"

// DayOfWeekInput maps accepted day-name inputs (lower-cased) to 0-based weekday index (Mon=0).
var DayOfWeekInput = map[string]int{
	"mon": 0, "monday": 0,
	"tue": 1, "tuesday": 1,
	"wed": 2, "wednesday": 2,
	"thu": 3, "thursday": 3,
	"fri": 4, "friday": 4,
	"sat": 5, "saturday": 5,
	"sun": 6, "sunday": 6,
}

// Reschedule
const (
	RescheduleModalPrefix      = "manage_reschedule_modal"
	RescheduleComponentPrefix  = "manage_reschedule"
	RescheduleModalTitle       = "Configure Schedule"
	RescheduleDayFieldID       = "reschedule_day"
	RescheduleDayLabel         = "Day of Week"
	RescheduleDayPlaceholder   = "e.g. Saturday or sat"
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
	RescheduleInvalidDay       = "Invalid day. Use a name like Monday, Tuesday... or abbreviation like mon, tue..."
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
	MeCommandName     = "me"
	MeCommandDesc     = "Your player profile and quick actions."
	MeHubMessage      = "Hey, <@%s>! What would you like to do?"
	MeCampaignsLabel  = "Campaigns"
	MeConfigLabel     = "Configuration"
	MeCampaignsHeader = "Campaigns"
	MeConfigHeader    = "Configuration"
	ControlPanelLabel = "Control Panel"
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
	AdminCommandName     = "admin"
	AdminCommandDesc     = "Mod/Admin panel."
	AdminNotStaff        = "Only mods or admins can use this command."
	DebugSurfaceDisabled = "This surface is disabled in production. Set DEV_MODE=true to re-enable."
	AdminHubMessage      = "Admin Panel:"
	AdminCampaignsLabel  = "Query Campaigns"
	AdminDMsLabel        = "DMs"
	AdminBroadcastLabel  = "Broadcast"
	AdminDatabaseLabel   = "Database"
	AdminSettingsLabel   = "Settings"
	AdminDiagLabel       = "Diagnostics"
)

// About (/moontracer)
const (
	AboutCommandName           = "about"
	AboutCommandDesc           = "About this bot."
	AboutCommandGitHubRepoLink = "https://github.com/framebuffers/moontracer"
	AboutCommandGitHubLabel    = "GitHub"
	HelpLabel                  = "Help"
	AboutCommandWebsite        = "https://framebuffer.cl/moontracer"
	AboutCommandBotDesc        = "Moontracer\n_a D&D campaign manager for players, DM and spectators!_"
	AboutCommandCopyright      = "(C) 2026 **Framebuffer**"
	AboutCommandLicense        = "AGPL-v3.0"
	AboutCommandHelp           = "Type `/help` for a list of commands."
	AboutCommandAwoo           = "awoo!"
	AboutCommandAttributions   = "Thanks to the D&D r/Chile Discord server for giving me the idea, letting me test the bot on their server and give me feedback to improve this bot."
)

// Navigation buttons
const (
	BackLabel = ""
	HomeLabel = "🏠"
)

// Hub button labels
const (
	MyCampaignsLabel     = "My Campaigns"
	NextSessionsLabel    = "Next Sessions"
	NotificationsLabel   = "Notifications"
	BrowseCampaignsLabel = "Browse Campaigns"
	MyProfileLabel       = HomeLabel
	NewCampaignLabel     = "New Campaign"
	AdminPanelLabel      = "Admin Panel"
)

// Manage campaign hub labels
const (
	ManagePlayersLabel  = "Players"
	ManageSessionsLabel = "Sessions"
	ManageSettingsLabel = "Settings"
	ManageDangerLabel   = "⚠️ Spicy Zone"
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
	FieldSlotsPlaceholder = "e.g. 4 (leave empty for unlimited)"
	FieldSynopsisLabel    = "Synopsis & Rules"
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
	NextSessionsPrefix = "next_sessions"
	NextSessionsHeader = "Upcoming sessions:"
	NextSessionsNone   = "You have no upcoming sessions."
)

// Player hub: Notifications
const (
	NotificationsPrefix = "notifications"
	NotificationsHeader = "Notification settings:"
	NotificationsNone   = "No notification preferences configured yet."

	NotifTogglePrefix       = "notif_toggle"
	NotifFieldAnnouncements = "announcements"
	NotifFieldSessions      = "sessions"
	NotifFieldInvitations   = "invitations"
	NotifLabelAnnouncements = "Announcements"
	NotifLabelSessions      = "Session Reminders"
	NotifLabelInvitations   = "Invitations"
	NotifLoadFailed         = "Failed to load notification settings."
	NotifUpdateFailed       = "Failed to update notification settings."
	NotifEnabledSuffix      = "%s: ON"
	NotifDisabledSuffix     = "%s: OFF"
)

// Admin hub: Campaign browser (all campaigns)
const (
	AdminCampaignsPrefix           = "admin_campaigns"
	AdminCampaignsHeader           = "All campaigns:"
	AdminCampaignsNone             = "No campaigns in the database."
	AdminCampaignSelectPlaceholder = "Pick a campaign for details..."
	AdminContactDMLabel            = "Contact DM"
)

// Admin hub: Broadcast
const (
	AdminBroadcastPrefix     = "admin_broadcast"
	AdminBroadcastModalID    = "modal_admin_broadcast"
	AdminBroadcastModalTitle = "Broadcast Message"
	AdminBroadcastFieldLabel = "Message"
	AdminBroadcastFieldID    = "broadcast_message"
	AdminBroadcastSuccess    = "Broadcast sent."
	AdminBroadcastSent       = "Broadcast sent to %d players."
	AdminBroadcastFailed     = "Failed to send broadcast."
	AdminBroadcastDMContent  = "**Broadcast from <@%s>:**\n\n%s"
)

// Admin hub: Database viewer
const (
	AdminDatabasePrefix = "admin_database"
	AdminDBCampaignLine = "**%s** (`%s`) — DM: <@%s> [%s]"
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

// Manage campaign: New Campaign from button (modal-from-component)
const (
	ManageNewCampaignPrefix = "manage_newcampaign"
)

// Manage campaign: Invite Player
const (
	ManageInviteLabel  = "Invite Player"
	ManageInvitePrefix = "manage_invite"

	ManageSetSessionLabel           = "Set Session"
	ManageRescheduleSessionLabel    = "Reschedule"
	ManageSetSessionPrefix          = "manage_set_session"
	ManageSetSessionModalID         = "modal_manage_set_session"
	ManageSetSessionModalTitle      = "Set Next Session"
	ManageSetSessionDateLabel       = "Date (YYYY-MM-DD)"
	ManageSetSessionDatePlaceholder = "2026-05-08"
	ManageSetSessionTimeLabel       = "Time UTC (HH:MM, 24h)"
	ManageSetSessionTimePlaceholder = "19:00"
	ManageSetSessionDateFieldID     = "session_date"
	ManageSetSessionTimeFieldID     = "session_time"
	ManageSetSessionInvalidDate     = "Invalid date format. Use YYYY-MM-DD."
	ManageSetSessionInvalidTime     = "Invalid time format. Use HH:MM (24h)."
	ManageSetSessionInPast          = "Cannot set a session in the past."
	ManageSetSessionSuccess         = "Next session for **%s** set to **%s** — %s."
	ManageSetSessionUpdateFailed    = "Failed to update next session."

	// Reschedule-specific (existing session → new date + reason).
	ManageRescheduleModalTitle        = "Reschedule Session"
	ManageSetSessionReasonFieldID     = "session_reason"
	ManageSetSessionReasonLabel       = "Reason for change (optional)"
	ManageSetSessionReasonPlaceholder = "e.g. DM unavailable this week"
	ManageSetSessionRescheduleThread  = "📅 Session rescheduled to **%s** — _%s_"
	ManageSetSessionRescheduleSuccess = "Session for **%s** rescheduled to **%s** — %s. Reason posted to thread."

	// Session reminder DM (sent ~1 hour before NextSession).
	ReminderContent          = "**Session Reminder: %s**\nYour next session starts in about 1 hour — **%s** (%s)"
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

// Session RSVP (buttons on reminder DMs)
const (
	RSVPAcceptPrefix     = "rsvp_accept"
	RSVPDeclinePrefix    = "rsvp_decline"
	RSVPAcceptLabel      = "✅ I'm Going!"
	RSVPDeclineLabel     = "❌ I'm Not Going"
	RSVPAcceptedPlayer   = "✅ Confirmed! The DM has been notified. May the RNG be with you!"
	RSVPDeclinedPlayer   = "❌ Noted. The DM has been notified. Have a good day!"
	RSVPDMNotifyAccept   = "✅ <@%s> confirmed assistance at **%s** — %s."
	RSVPDMNotifyDecline  = "❌ <@%s> won't be coming for **%s** — %s."
	RSVPAlreadyResponded = "You've already responded for this session. If you changed your mind, talk to your DM"
	RSVPCampaignGone     = "This campaign is no longer active."
)

// Timezone preference
const (
	TimezoneLabel             = "Timezone"
	TimezonePrefix            = "set_timezone"
	TimezoneSelectID          = "timezone_select"
	TimezoneHeader            = "**Set your timezone**\nTimes will be shown in your local time.\nCurrent: **%s**"
	TimezoneSelectPlaceholder = "Select your timezone…"
	TimezoneSuccess           = "Timezone set to **%s**."
	TimezoneInvalid           = "Unknown timezone. Please select from the list."
)
