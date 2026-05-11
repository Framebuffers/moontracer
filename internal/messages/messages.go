package messages

/*

	Messages:
		Every single string inside Moontracer is described here.
		This is such that there is a single source of truth for all string values inside the Bot.
		If the dev wants to change any string, they can change it here.

	Organization:
		Within each scope, constants are split into two groups:
		  - identifiers: command names, custom IDs, field IDs, component prefixes, internal log
		    prefixes. These are interface contracts or internal keys. Do not translate these.
		  - user-facing: messages, labels, descriptions, placeholders, and success/error copy shown
		    to users. These can be translated.

*/

// Generic
const (
	// identifiers
	BotVersion = "v0.12.3"
)
const (
	// user-facing
	GenericErrorMessage      = ":no_entry_sign: Something went wrong."
	InvalidButtonDataMessage = ":warning: Invalid button data."
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
	// input format contracts. changing these breaks modal parsing
	DateInputFormat     = "2006-01-02"       // YYYY-MM-DD (ISO 8601 date)
	TimeInputFormat     = "15:04"            // HH:MM (24h)
	DateTimeInputFormat = "2006-01-02 15:04" // combined for ParseInLocation
)
const (
	// user-facing display formats. swap per locale for localized day/month names
	SessionTimeFormat = "Mon 2 Jan 2006 15:04" // long form, includes year
	SessionListFormat = "Mon 2 Jan 15:04"      // compact, year implied
)

// Command names and descriptions
const (
	// identifiers
	PingCommandName        = "ping"
	AwooCommandName        = "awoo"
	RegisterCommandName    = "register"
	CampaignCommandName    = "campaign"
	MyCampaignsCommandName = "mycampaigns"
	AddPlayerCommandName   = "add_player"
	TagCommandName         = "tag"
)
const (
	// user-facing
	PingCommandDesc        = "Replies with pong!"
	AwooCommandDesc        = "do a heccin awoo."
	RegisterCommandDesc    = "Register as a player so you can join and create campaigns."
	CampaignCommandDesc    = "Show campaign details."
	MyCampaignsCommandDesc = "List the campaigns you're part of."
	AddPlayerCommandDesc   = "Adds a new player to a Campaign."
	TagCommandDesc         = "Campaign tag to look up."
)

// Registration
const (
	// internal log prefixes
	RegistrationCheckError  = "register: error checking registration: "
	RegistrationInsertError = "register: error inserting player: "
	QuickRegisterPrefix     = "quick_register"
)
const (
	// user-facing
	RegisterButtonLabel        = "Register"
	NotRegisteredMessage       = ":warning: You need to register first."
	AlreadyRegisteredMessage   = ":information_source: You are already registered!"
	RegistrationFailureMessage = ":no_entry_sign: Failed to register. Please try again later."
	RegistrationSuccessMessage = ":wolf: Welcome, <@%s>! You are now registered."
)

// Campaign lookup
const (
	// internal log prefixes
	CampaignFetchError      = "campaign: error fetching campaign %s: "
	PlayerFetchErrorMessage = "models.GetCampaignPlayers(): Error fetching players: "
)
const (
	// user-facing
	CampaignNotFoundMessage         = ":information_source: Campaign not found."
	CampaignArchivedFooter          = ":information_source: This campaign is archived. It is read-only now."
	CampaignLoadFailureErrorMessage = ":no_entry_sign: Failed to load campaign."
	CampaignPlayersLoadError        = ":no_entry_sign: Failed to load campaign players."
)

// Campaign creation
const (
	// internal log prefix
	CampaignCreationFailureErrorMessage = "campaign.CreateCampaign(): error creating campaign: "
)
const (
	// user-facing
	SlotCountMismatchErrorMessage              = ":no_entry_sign: Invalid slot count. Capacity must be a positive number. Leave the field empty for unlimited."
	CampaignAndRegistrationFailureErrorMessage = ":no_entry_sign: Failed to create campaign. Make sure you are registered."
	CampaignCreationMessage                    = ":white_check_mark: You just created a new campaign: "
	CampaignStaffNotifyFailureMessage          = ":warning: Could not notify staff members to ask for approval of this Campaign."
	CampaignApprovalRequestMessage             = ":information_source: New campaign **%s** by <@%s> needs approval."
)

// Campaign join
const (
	// internal log prefix
	InsertPlayerErrorMessage = "db.Insert(): Error inserting Campaign Player: "
)
const (
	// user-facing
	CampaignClosedMessage          = ":warning: This campaign is not open for new players."
	PlayerBannedMessage            = ":no_entry: You are banned from this campaign."
	PlayerAlreadyOnCampaignMessage = ":information_source: You are already in this campaign."
	CampaignFullMessage            = ":warning: This campaign is full."
	PlayerFailedToJoinMessage      = ":no_entry_sign: Failed to join campaign."
	PlayerJoinedCampaignMessage    = ":wolf: You have joined **%s**!"

	// Westmarch session-capacity tripwire (FCFS soft alert).
	WestmarchOverCapacityDMAlert      = ":warning: **INFO:** <@%s> just joined westmarch **%s**. Roster is now %d active player(s). Session capacity is %d. They've been admitted; bring an extra seat or trim attendance for the next session."
	WestmarchOverCapacityPlayerNotice = ":wolf: You're in **%s**! \n:warning: Warning!: this westmarch's session capacity (%d) is already met, so the DM has been notified. Talk to your DM for more help."
)

// Campaign leave
const (
	// internal log prefix
	LeavingCampaignErrorMessage = "models.RemoveCampaignPlayer(): error removing player: "
)
const (
	// user-facing
	MasterIsLeavingCampaignErrorMessage = ":warning: You are the DM. You cannot leave your own campaign."
	FailedToLeaveCampaignErrorMessage   = ":no_entry_sign: Failed to leave campaign."
	PlayerLeftCampaignMessage           = ":information_source: You have left **%s**."
)

// Campaign toggle
const (
	// internal log prefix
	CampaignUpdateErrorMessage = "db.Update(): error updating campaign: "
)
const (
	// user-facing
	MasterCanToggleStatusErrorMessage = ":information_source: Only the DM can toggle campaign status."
	CampaignStatusMessage             = "**%s** is now **%s**."
)

// My campaigns (all user-facing)
const (
	NoCampaignsMessage   = ":information_source: You are not in any campaigns yet!"
	MyCampaignsLoadError = ":warning: Failed to load your campaigns."
)

// Campaign embed UI labels
const (
	// identifiers
	EmbedColor = 0x5865F2
)
const (
	// user-facing
	OpenCampaignLabel          = ":blue_circle: Set as Open Campaign"
	ClosedCampaignLabel        = ":red_circle: Set as Closed Campaign"
	LeaveCampaignLabel         = ":door: Leave Campaign"
	JoinCampaignLabel          = ":green_circle: Join Campaign"
	ClosedStatusLabel          = ":no_entry_sign: Closed"
	OpenStatusLabel            = ":green_circle: Open"
	ArchivedStatusLabel        = ":yellow_circle: Archived"
	CampaignLabel              = ":european_castle: Campaign"
	CampaignTypeOneShotLabel   = ":book: One-shot"
	CampaignTypeWestmarchLabel = ":shield: Westmarch"
	NoneLabel                  = "None"
	NoBooksSpecifiedLabel      = ":warning: No books specified"
)

// New campaign modal
const (
	// identifiers
	NewCampaignModalCustomID = "modal_campaign_create"
	NewCampaignCommandName   = "newcampaign"
)
const (
	// user-facing
	NewCampaignModalError  = "newcampaign: error opening modal: "
	NewCampaignModalTitle  = ":european_castle: Create a New Campaign"
	NewCampaignCommandDesc = "Create a new campaign (you will be the DM!)"
)

// New campaign modal field IDs (all identifiers)
const (
	FieldNameID        = "name"
	FieldTagID         = "tag"
	FieldDescriptionID = "description"
	FieldEditionID     = "edition"
	FieldSlotsID       = "slots"
)

// New campaign modal labels (all user-facing)
const (
	FieldNameLabel        = "Name"
	FieldTagLabel         = "Tag"
	FieldDescriptionLabel = "Description"
	FieldEditionLabel     = "Edition"
	FieldSlotsLabel       = "Player Slots"
)

// New campaign modal placeholders (all user-facing)
const (
	FieldNamePlaceholder        = "e.g. Curse of Strahd"
	FieldTagPlaceholder         = "e.g. curse-of-strahd (short, no spaces)"
	FieldDescriptionPlaceholder = "Describe your campaign setting and premise..."
	FieldEditionPlaceholder     = "e.g. 5e, 3.5e, PF2e"
)

// Add player (all user-facing)
const (
	AddPlayerNotDMOrModMessage   = ":information_source: You must be the DM of this campaign to add players."
	AddPlayerTargetNotRegistered = ":information_source: That user is not registered. They need to `/register` first."
	AddPlayerAlreadyInCampaign   = ":warning: That player is already in this campaign."
	AddPlayerCampaignFullMessage = ":no_entry_sign: This campaign is full."
	AddPlayerFailureMessage      = ":warning: Failed to add player to campaign."
	AddPlayerSuccessMessage      = ":wolf: Added <@%s> to **%s**!"
)

// Ban
const (
	// identifiers
	BanCommandName = "ban"
)
const (
	// user-facing
	BanCommandDesc          = "Globally ban a player from the server."
	BanCannotBanSelf        = ":warning: You cannot ban yourself."
	BanInsufficientRole     = ":warning: You cannot ban someone of equal or higher role."
	BanTargetNotFound       = ":information_source: That player is not registered."
	BanPlayerNotInCampaign  = ":warning: The player ID is not valid for this campaign, or it does not belong to it."
	BanTargetAlreadyBanned  = ":information_source: That player is already banned."
	BanFailureMessage       = ":no_entry_sign: Failed to ban player."
	BanSuccessMessage       = ":information_source: Banned <@%s>. \nThey can no longer interact with the bot."
	BanSuccessReasonMessage = ":information_source: Banned <@%s>: %s"
)

// Unban
const (
	// identifiers
	UnbanCommandName = "unban"
)
const (
	// user-facing
	UnbanCommandDesc     = "Lift a global ban from a player."
	UnbanTargetNotFound  = ":information_source: That player is not registered."
	UnbanTargetNotBanned = ":information_source: That player is not banned."
	UnbanFailureMessage  = ":no_entry_sign: Failed to unban player."
	UnbanSuccessMessage  = ":information_source: Unbanned <@%s>. They can interact with the bot again."
)

// Campaign archival
const (
	// identifiers: command name and audit-log reasons stored in DB (do not translate!)
	AbandonCommandName      = "abandon"
	AbandonReasonDM         = "DM abandoned"
	AbandonReasonLeftServer = "DM left server"
)
const (
	// user-facing
	CampaignArchivedMessage = ":information_source: This campaign has been archived and can no longer be modified."
	AbandonCommandDesc      = "Archive your campaign permanently. Only the DM can do this."
	AbandonNotDMMessage     = ":information_source: Only the DM of this campaign can abandon it."
	AbandonFailureMessage   = ":no_entry_sign: Failed to archive campaign."
	AbandonSuccessMessage   = ":information_source: Campaign **%s** has been archived. It is now an immutable record, and cannot be recovered."
)

// Manage campaigns
const (
	// identifiers
	ManageCampaignsCommandName = "managecampaigns"
)
const (
	// user-facing
	ManageCampaignsCommandDesc = "Manage campaigns you run as DM."
	ManageCampaignsLabel       = "Manage Campaigns"
	ManageNoDMCampaigns        = ":warning: You are not the DM of any campaigns."
	ManageNotAuthorized        = ":information_source: You must be the DM of this campaign to manage it."
	ManageCampaignNotFound     = ":information_source: Campaign not found."
	ManageDeleteSuccess        = ":information_source: Campaign **%s** has been deleted."
	ManageDeleteFailure        = ":no_entry_sign: Failed to delete campaign."
	ManageBanNoMembers         = ":information_source: This campaign has no members to ban."
	ManageCampaignBanSuccess   = ":information_source: Banned <@%s> from **%s**."
	ManageCampaignBanFailure   = ":no_entry_sign: Failed to ban player from campaign."
	ManageSelectMember         = "Select a member to ban from **%s**:"
	ManageBanSelectPlaceholder = "Select a member..."
	ManageCampaignHeader       = "Managing **%s**:"
	ManageInviteSelectPrompt   = "Select a player to invite to **%s**:"
	MyCampaignListLine         = "**%s** — %s (%s)"
	ManageCampaignListLine     = "**%s** — %s"
)

// Token upload
const (
	// identifiers — token gallery
	TokenGallerySelectPrefix       = "token_gallery_select"
	TokenGalleryAssignPrefix       = "token_gallery_assign"
	TokenGalleryAssignSelectPrefix = "token_gallery_assign_select"
	TokenDeletePromptPrefix        = "token_delete_prompt"
	TokenDeleteConfirmPrefix       = "token_delete_confirm"
)
const (
	// user-facing — token gallery
	TokensLabel                         = "Tokens"
	TokenGalleryHeader                  = "Your tokens:"
	TokenGalleryNone                    = ":information_source: You have no tokens yet! Use `/newtoken` to create one."
	TokenNotAssigned                    = ":information_source: This token is not assigned to any campaign"
	TokenGallerySelectPlaceholder       = "Select a token to manage..."
	TokenDeleteLabel                    = ":wastebasket: Delete"
	TokenDeleteConfirmMsg               = ":warning: Are you sure you want to delete **%s**? This cannot be undone."
	TokenDeleteConfirmLabel             = ":wastebasket: Yes, Delete"
	TokenDeleteCancelLabel              = ":x: Cancel"
	TokenDeleteSuccess                  = ":information_source: Token **%s** deleted."
	TokenDeleteFailed                   = ":warning: Failed to delete token."
	TokenGalleryAssignPrompt            = ":information_source: Assign **%s** to a campaign:"
	TokenGalleryAssignSelectPlaceholder = "Select a campaign..."
)

// Token upload
const (
	// identifiers
	TokenUploadCommandName      = "newtoken"
	TokenUploadSourceOptName    = "source"
	TokenUploadFrameOptName     = "frame"
	TokenUploadColorOptName     = "color"
	TokenApplyPrefix            = "token_apply"
	TokenApplyModalPrefix       = "token_apply_modal"
	TokenDiscardPrefix          = "token_discard"
	TokenPostcreateSelectPrefix = "player_token_postcreate"
	TokenSkipPrefix             = "player_token_skip"
	TokenNameFieldID            = "token_name"
)
const (
	// user-facing
	TokenUploadCommandDesc           = "Create a player token from a photo, with a frame image or a solid color border."
	TokenUploadSourceOptDesc         = "Your photo (JPG/PNG)."
	TokenUploadFrameOptDesc          = "Frame/border image (PNG with transparency). Cannot be used with color."
	TokenUploadColorOptDesc          = "Border color as hex (e.g. ff3a7c). Cannot be used with frame."
	TokenUploadNeedOneOf             = ":no_entry_sign: Provide either a frame image or a color — not both, not neither."
	TokenUploadNotImage              = ":warning: Both files must be images (JPEG or PNG)."
	TokenUploadTooLarge              = ":warning: Each file must be under 8 MiB."
	TokenUploadProcessFailed         = ":no_entry_sign: Failed to process your token. Please try again."
	TokenUploadPreviewContent        = ":information_source: Here's a preview of your token. Apply it to your profile or discard."
	TokenApplyLabel                  = ":white_check_mark: Apply"
	TokenDiscardLabel                = ":x: Discard"
	TokenPostcreateSelectPlaceholder = "Assign to a campaign..."
	TokenSkipLabel                   = "Skip for now"
	TokenPostcreateHeader            = ":green_check_mark: **%s** saved! Assign it to one of your campaigns, or skip."
	TokenPostcreateAssigned          = ":wolf: Token assigned to **%s**!"
	TokenSavedNoAssign               = ":information_source: Token saved. You can assign it later from your campaign card."
	TokenNameModalTitle              = "Name Your Token"
	TokenNameFieldLabel              = "Character Name"
	TokenNameFieldPlaceholder        = "e.g. Soft Doggo"
	TokenApplySuccess                = ":white_check_mark: Token saved as **%s**!"
	TokenDiscardSuccess              = ":information_source: Token discarded."
	TokenApplyFailed                 = ":warning: Failed to save token. Please try again."
)

// Campaign cover / upload
const (
	// identifiers
	CampaignUploadCommandName     = "campaignupload"
	CampaignUploadKindOptName     = "kind"
	CampaignUploadKindCoverChoice = "Cover"
	CampaignUploadCampaignOptName = "campaign"
	CampaignUploadImageOptName    = "image"
)
const (
	// user-facing
	CampaignUploadCommandDesc     = "Upload an image for one of your campaigns."
	CampaignUploadKindOptDesc     = "What kind of image to upload."
	CampaignUploadCampaignOptDesc = "The campaign to upload an image for."
	CampaignUploadImageOptDesc    = "Image file (JPEG/PNG/WebP, up to 8 MiB)."
	CampaignUploadNotDM           = ":information_source: Only the DM of this campaign can change its cover."
	CampaignUploadNotImage        = ":warning: That file doesn't look like an image. Try JPEG, PNG, or WebP."
	CampaignUploadTooLarge        = ":warning: Image is too large. Max 8 MiB."
	CampaignUploadMissingAttach   = ":information_source: No image attached. Attach a file to the `image` option."
	CampaignUploadFailure         = ":warning: Failed to save cover. Please try again."
	CampaignUploadSuccess         = ":white_check_mark: Cover set for **%s**. [View](%s)"
	SetCoverButtonLabel           = "Set Cover"
	SetCoverInstructions          = "Use `/campaignupload kind:Cover campaign:<name> image:<file>` to set a cover for this campaign."
)

// Manage campaigns: button labels (all user-facing)
const (
	ManageDeleteLabel           = ":wastebasket: Delete"
	ManageBanLabel              = ":no_entry_sign: Ban Member"
	ManageAnnounceLabel         = ":european_castle: New Session"
	ManageRescheduleLabel       = ":calendar: Configure Schedule"
	ManageCampaignButtonLabel   = ":gear: Manage"
	ManageDownloadTokensLabel   = ":arrow_down: Player Tokens"
	ManageDownloadTokensNone    = ":information_source: No players in this campaign have tokens assigned."
	ManageDownloadTokensContent = "Tokens for **%s** (%d player(s)):"
)

// Set campaign role
const (
	// identifiers
	SetCampaignRoleCommandName = "setcampaignrole"
	SetRoleFieldName           = "role"
)
const (
	// user-facing
	SetCampaignRoleCommandDesc = "Link a Discord role to a campaign (creates one if it doesn't exist)."
	SetRoleFieldDesc           = "Name of the Discord role to link."
	SetRoleNotDMOrMod          = ":warning: You must be the DM of this campaign to set its role."
	SetRoleSuccess             = ":white_check_mark: Linked role **%s** to campaign **%s**."
	SetRoleCreateFailed        = ":warning: Failed to create Discord role."
	SetRoleUpdateFailed        = ":warning: Failed to update campaign."
)

// Campaign approval
const (
	// identifiers
	CampaignApprovePrefix     = "campaign_approve"
	CampaignDenyPrefix        = "campaign_deny"
	CampaignDenyModalPrefix   = "campaign_deny_modal"
	CampaignDenyReasonFieldID = "deny_reason"
)
const (
	// user-facing
	ApproveButtonLabel            = ":white_check_mark: Approve"
	DenyButtonLabel               = ":no_entry_sign: Deny"
	CampaignApprovedMessage       = ":wolf: Campaign **%s** has been approved."
	CampaignDeniedMessage         = ":no_entry_sign: Campaign **%s** has been denied and deleted."
	CampaignApproveNotModError    = ":warning: Only mods or admins can approve campaigns."
	CampaignApproveNotFound       = ":information_source: Campaign not found or already processed."
	CampaignApproveError          = ":warning: Failed to process campaign approval."
	CampaignDenyModalTitle        = ":no_entry_sign: Deny Campaign"
	CampaignDenyReasonLabel       = "Reason"
	CampaignDenyReasonPlaceholder = "Why is this campaign being denied?"
	CampaignDeniedDMMessage       = ":no_entry_sign: Your campaign **%s** has been denied. Reason: %s"
	CampaignApprovedDMMessage     = ":wolf: Your campaign **%s** has been approved!"
	CampaignApprovedStatusMessage = ":white_check_mark: Approved campaign **%s**."
	CampaignDeniedStatusMessage   = ":no_entry_sign: Denied campaign **%s**. Reason: %s"
	CampaignDenyPendingMessage    = ":information_source: Campaign **%s**: denial in progress..."
)

// Announce
const (
	// identifiers
	AnnounceModalPrefix     = "manage_announce_modal"
	AnnounceComponentPrefix = "manage_announce"
	AnnounceFieldID         = "announce_message"
)
const (
	// user-facing
	AnnounceModalTitle       = "New Session Announcement"
	AnnounceFieldLabel       = "Message"
	AnnounceFieldPlaceholder = "Type your announcement to all campaign members..."
	AnnounceSentMessage      = ":speaking_head: Announcement sent to %d members of **%s**."
	AnnounceNoMembers        = ":information_source: This campaign has no members to announce to."
	AnnounceError            = ":warning: Failed to send announcement."
	AnnouncePostedToThread   = ":speaking_head: Announcement posted to the **%s** announcements thread."
	AnnounceThreadContent    = "%s**:speaking_head:Announcement from <@%s>:**\n\n%s"
	AnnounceDMContent        = "**[%s]** :speaking_head: Announcement from <@%s>:\n\n%s"
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
	// identifiers
	RescheduleModalPrefix     = "manage_reschedule_modal"
	RescheduleComponentPrefix = "manage_reschedule"
	RescheduleDayFieldID      = "reschedule_day"
	RescheduleTimeFieldID     = "reschedule_time"
	RescheduleDurFieldID      = "reschedule_duration"
	RescheduleFreqFieldID     = "reschedule_freq"
)
const (
	// user-facing
	RescheduleModalTitle       = "Configure Schedule"
	RescheduleDayLabel         = "Day of Week"
	RescheduleDayPlaceholder   = "e.g. Saturday or sat"
	RescheduleTimeLabel        = "Start Time (HH:MM UTC)"
	RescheduleTimePlaceholder  = "e.g. 19:00"
	RescheduleDurLabel         = "Duration (hours)"
	RescheduleDurPlaceholder   = "e.g. 3"
	RescheduleFreqLabel        = "Frequency (weekly, biweekly, monthly, quarterly, yearly)"
	RescheduleFreqPlaceholder  = "e.g. weekly"
	RescheduleSuccess          = ":white_check_mark: Schedule updated for **%s**: %s %s UTC (%sh), %s."
	RescheduleInvalidDay       = ":warning: Invalid day. Use a name like Monday, Tuesday... or abbreviation like mon, tue..."
	RescheduleInvalidTime      = ":warning: Invalid time format. Use HH:MM (e.g. 19:00)."
	RescheduleInvalidDuration  = ":warning: Invalid duration. Enter a number (e.g. 3)."
	RescheduleInvalidFrequency = ":warning: Invalid frequency. Use: weekly, biweekly, monthly, quarterly, yearly."
	RescheduleError            = ":no_entry_sign: Failed to update schedule."
)

// Campaign database (debug)
const (
	// identifiers
	CampaignDBCommandName = "campaigndatabase"
)
const (
	// user-facing
	CampaignDBCommandDesc = "Show all campaigns in the database (staff only)."
	CampaignDBEmpty       = ":information_source: No campaigns in the database."
	CampaignDBNotStaff    = ":information_source: Only mods or admins can use this command."
)

// Help command
const (
	// identifiers
	HelpCommandName = "help"
)
const (
	// user-facing
	HelpCommandDesc = "Get a list of all available commands."
)

// Player hub (/me)
const (
	// identifiers
	MeCommandName = "me"
)
const (
	// user-facing
	MeCommandDesc     = "Your player profile and quick actions."
	MeHubMessage      = "Hey, <@%s>! What would you like to do?"
	MeCampaignsLabel  = ":european_castle: Campaigns"
	MeConfigLabel     = ":gear: Configuration"
	MeCampaignsHeader = ":european_castle: Campaigns"
	MeConfigHeader    = ":gear: Configuration"
	ControlPanelLabel = ":control_knobs: Control Panel"
)

// Browse campaigns (/campaigns)
const (
	// identifiers
	CampaignsCommandName = "campaigns"
)
const (
	// user-facing
	CampaignsCommandDesc       = "Browse all available campaigns."
	CampaignsFilterPlaceholder = "Filter by format..."
	CampaignsSelectPlaceholder = "Select a campaign..."
	CampaignsNoneAvailable     = ":information_source: No campaigns match this filter."
	CampaignsFilterAll         = "All"
	CampaignsFilterCampaign    = "Campaigns"
	CampaignsFilterOneshot     = "One-shots"
	CampaignsFilterWestmarch   = "Westmarches"
)

// Search (/search)
const (
	// identifiers
	SearchCommandName = "search"
	SearchOptionName  = "name"
)
const (
	// user-facing
	SearchCommandDesc = "Search for a campaign by name."
	SearchOptionDesc  = "Campaign name to search for."
	SearchNoResults   = ":information_source: No campaigns found matching that name."
)

// Admin hub (/admin)
const (
	// identifiers
	AdminCommandName = "admin"
)
const (
	// user-facing
	AdminCommandDesc     = "Mod/Admin panel."
	AdminNotStaff        = ":information_source: Only mods or admins can use this command."
	DebugSurfaceDisabled = ":information_source: This surface is disabled in production. Set DEV_MODE=true to re-enable."
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
	// identifiers
	AboutCommandName           = "about"
	AboutCommandGitHubRepoLink = "https://github.com/framebuffers/moontracer"
	AboutCommandWebsite        = "https://framebuffer.cl/moontracer"
	AboutCommandLicense        = "AGPL-v3.0"
)
const (
	// user-facing
	AboutCommandDesc         = "About this bot."
	AboutCommandGitHubLabel  = "GitHub"
	AboutLabel               = "About"
	HelpLabel                = "Help"
	AboutCommandBotDesc      = "_a D&D campaign manager for players, DM and spectators!_"
	AboutCommandCopyright    = "(C) 2026 **Framebuffer**"
	AboutCommandHelp         = "Type `/help` for a list of commands."
	AboutCommandAwoo         = "awoo!"
	AboutCommandAttributions = "-#Thanks to the D&D r/Chile Discord server for giving me the idea, letting me test the bot on their server and give me feedback to improve this bot."
)

// Navigation buttons (all user-facing)
const (
	BackLabel = ""
	HomeLabel = "🏠"
)

// Hub button labels (all user-facing)
const (
	MyCampaignsLabel     = "My Campaigns"
	NextSessionsLabel    = "Next Sessions"
	NotificationsLabel   = "Notifications"
	BrowseCampaignsLabel = "Browse Campaigns"
	MyProfileLabel       = HomeLabel
	NewCampaignLabel     = "New Campaign"
	AdminPanelLabel      = "Admin Panel"
)

// Manage campaign hub labels (all user-facing)
const (
	ManagePlayersLabel   = "Players"
	ManageSessionsLabel  = "Sessions"
	ManageSettingsLabel  = "Settings"
	ManageDangerLabel    = ":warning: Spicy Zone"
	ManageOpenLabel      = ":green_circle: Open Campaign"
	ManageCloseLabel     = ":red_circle: Close Campaign"
	CampaignAutoClosedDM = ":information_source: **%s** has been automatically closed — all %d slots are filled."
)

// Manage campaign: Links
const (
	// identifiers
	ManageLinksPrefix  = "manage_links"
	ManageLinksModalID = "modal_manage_links"
)
const (
	// user-facing
	ManageLinksLabel                = "Links"
	ManageLinksModalTitle           = "Campaign Links"
	ManageLinksVTTLabel             = "VTT Link"
	ManageLinksVTTPlaceholder       = "https://owlbear.rodeo/... or https://app.roll20.net/..."
	ManageLinksResourcesLabel       = "Session Resources (one URL per line)"
	ManageLinksResourcesPlaceholder = "https://drive.google.com/...\nhttps://example.com/map"
	ManageLinksSuccess              = ":white_check_mark: Links updated for **%s**."
	ManageLinksEmbedTitle           = ":chains: Links"
	ManageReminderLinks             = "\n\n:chains: **Links for tonight:**"
	ManageReminderVTT               = "\nVTT: %s"
	ManageReminderSheets            = "\nSheets: %s"
	ManageReminderResource          = "\n• %s"
)

// Manage campaign: additional buttons
const (
	// identifiers
	ManageSetRolePrefix        = "manage_role"
	ManageArchivePrefix        = "manage_archive"
	ManageDownloadTokensPrefix = "manage_download_tokens"
	ManageSetRoleModalID       = "modal_manage_role"
	ManageDeleteConfirmID      = "manage_delete_confirm"
	ManageArchiveConfirmID     = "manage_archive_confirm"
	ManageArchiveCancelID      = "manage_archive_cancel"
)
const (
	// user-facing
	ManageSetRoleLabel = "Set Role"
	ManageArchiveLabel = "Archive"

	// Set Role modal
	ManageSetRoleModalTitle = "Link Discord Role"
	ManageSetRoleFieldLabel = "Role name (creates if it doesn't exist)"
	ManageSetRoleSuccess    = ":white_check_mark: Linked role **%s** to campaign **%s**."
	ManageSetRoleFailed     = ":no_entry_sign: Failed to set role."

	// Delete confirmation + handler
	ManageDeleteConfirm      = ":warning: Are you sure you want to delete **%s**? This is permanent and cannot be undone. All members will be removed."
	ManageDeleteConfirmLabel = "Yes, Delete"
	ManageDeleteCancelLabel  = "Cancel"

	// Archive confirmation + handler
	ManageArchiveConfirm      = ":warning: Are you sure you want to archive **%s**? This is permanent and cannot be undone."
	ManageArchiveConfirmLabel = ":white_check_mark: Yes, Archive"
	ManageArchiveCancelLabel  = ":x: Cancel"
	ManageArchiveSuccess      = ":information_source: Campaign **%s** has been archived. It is now an immutable record."
	ManageArchiveFailed       = ":no_entry_sign: Failed to archive campaign."
)

// New campaign config (post-modal dropdowns)
const (
	// identifiers
	NewCampaignBookPrefix   = "newcampaign_book"
	NewCampaignFormatPrefix = "newcampaign_format"
	NewCampaignSubmitPrefix = "newcampaign_submit"
	NewCampaignCancelPrefix = "newcampaign_cancel"
)
const (
	// user-facing
	NewCampaignBookPlaceholder   = "Select a game system..."
	NewCampaignFormatPlaceholder = "Select a format..."
	NewCampaignConfigMessage     = ":information_source: Campaign **%s** created (pending setup).\n\nSelect a game system and format, then submit for approval. You can set a cover image and links from the campaign settings after approval."
	NewCampaignSubmitLabel       = ":white_check_mark: Submit for Approval"
	NewCampaignCancelLabel       = ":x: Cancel"
	NewCampaignSubmittedMessage  = ":white_check_mark: Campaign **%s** has been submitted for approval!"
	NewCampaignCancelledMessage  = ":information_source: Campaign creation cancelled."

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

// Campaign Modal (all user-facing)
const (
	FieldSlotsPlaceholder = "e.g. 4 (leave empty for unlimited)"
	FieldSynopsisLabel    = "Synopsis & Rules"
)

// Select menu placeholders + content prefixes for /mycampaigns and /managecampaigns (all user-facing)
const (
	MyCampaignsPlaceholder     = "Select a campaign..."
	ManageCampaignsPlaceholder = "Select a campaign to manage..."
	MyCampaignsListHeader      = ":european_castle: Your campaigns:\n"
	ManageCampaignsListHeader  = ":european_castle: Your campaigns (DM):\n"
)

// Select menu CustomIDs (all identifiers)
const (
	CampaignSelectPrefix      = "campaign_select"
	MyCampaignSelectPrefix    = "mycampaign_select"
	ManageSelectPrefix        = "manage_select"
	CampaignsFilterPrefix     = "campaigns_filter"
	AdminCampaignSelectPrefix = "admin_campaign_select"
)

// Player hub: Next Sessions
const (
	// identifiers
	NextSessionsPrefix = "next_sessions"
)
const (
	// user-facing
	NextSessionsHeader = "Upcoming sessions:"
	NextSessionsNone   = ":information_source: You have no upcoming sessions."
)

// Player hub: Notifications
const (
	// identifiers
	NotificationsPrefix     = "notifications"
	NotifTogglePrefix       = "notif_toggle"
	NotifFieldAnnouncements = "announcements"
	NotifFieldSessions      = "sessions"
	NotifFieldInvitations   = "invitations"
)
const (
	// user-facing
	NotificationsHeader     = ":envelope: Notification settings:"
	NotificationsNone       = ":information_source: No notification preferences configured yet."
	NotifLabelAnnouncements = ":speaking_head: Announcements"
	NotifLabelSessions      = ":calendar: Session Reminders"
	NotifLabelInvitations   = ":envelope: Invitations"
	NotifLoadFailed         = ":no_entry_sign: Failed to load notification settings."
	NotifUpdateFailed       = ":no_entry_sign: Failed to update notification settings."
	NotifEnabledSuffix      = "%s: ON"
	NotifDisabledSuffix     = "%s: OFF"
)

// Admin hub: Campaign browser (all campaigns)
const (
	// identifiers
	AdminCampaignsPrefix = "admin_campaigns"
)
const (
	// user-facing
	AdminCampaignsHeader           = "All campaigns:"
	AdminCampaignsNone             = ":information_source: No campaigns in the database."
	AdminCampaignSelectPlaceholder = "Pick a campaign for details..."
	AdminContactDMLabel            = ":mailbox_with_mail: Contact DM"
)

// Admin hub: Broadcast
const (
	// identifiers
	AdminBroadcastPrefix  = "admin_broadcast"
	AdminBroadcastModalID = "modal_admin_broadcast"
	AdminBroadcastFieldID = "broadcast_message"
)
const (
	// user-facing
	AdminBroadcastModalTitle = "Broadcast Message"
	AdminBroadcastFieldLabel = "Message"
	AdminBroadcastSuccess    = ":white_check_mark: Broadcast sent."
	AdminBroadcastSent       = ":white_check_mark: Broadcast sent to %d players."
	AdminBroadcastFailed     = ":no_entry_sign: Failed to send broadcast."
	AdminBroadcastDMContent  = "**:speaking_head: Broadcast from <@%s>:**\n\n%s"
)

// Admin hub: Database viewer
const (
	// identifiers
	AdminDatabasePrefix = "admin_database"
)
const (
	// user-facing
	AdminDBCampaignLine = "**%s** (`%s`) — DM: <@%s> [%s]"
)

// Admin hub: Settings
const (
	// identifiers
	AdminSettingsPrefix = "admin_settings"
)
const (
	// user-facing
	AdminSettingsHeader = "Bot settings:"
)

// Admin hub: Diagnostics
const (
	// identifiers
	AdminDiagPrefix = "admin_diag"
)

// Manage campaign: New Campaign from button
const (
	// identifiers
	ManageNewCampaignPrefix = "manage_newcampaign"
)

// Manage campaign: Invite Player
const (
	// identifiers
	ManageInvitePrefix            = "manage_invite"
	ManageSetSessionPrefix        = "manage_set_session"
	ManageSetSessionModalID       = "modal_manage_set_session"
	ManageSetSessionDateFieldID   = "session_date"
	ManageSetSessionTimeFieldID   = "session_time"
	ManageSetSessionReasonFieldID = "session_reason"
	ManageInviteSelectPrefix      = "manage_invite_select"
	InviteAcceptPrefix            = "campaign_invite_accept"
	InviteDeclinePrefix           = "campaign_invite_decline"
)
const (
	// user-facing
	ManageInviteLabel               = "Invite Player"
	ManageSetSessionLabel           = "Set Session"
	ManageRescheduleSessionLabel    = "Reschedule"
	ManageSetSessionModalTitle      = "Set Next Session"
	ManageSetSessionDateLabel       = "Date (YYYY-MM-DD)"
	ManageSetSessionDatePlaceholder = "2026-05-08"
	ManageSetSessionTimeLabel       = "Time UTC (HH:MM, 24h)"
	ManageSetSessionTimePlaceholder = "19:00"
	ManageSetSessionInvalidDate     = ":warning: Invalid date format. Use YYYY-MM-DD."
	ManageSetSessionInvalidTime     = ":warning: Invalid time format. Use HH:MM (24h)."
	ManageSetSessionInPast          = ":warning: Cannot set a session in the past."
	ManageSetSessionSuccess         = ":white_check_mark: Next session for **%s** set to **%s** — %s."
	ManageSetSessionUpdateFailed    = ":no_entry_sign: Failed to update next session."

	// Reschedule-specific (existing session -> new date + reason).
	ManageRescheduleModalTitle        = "Reschedule Session"
	ManageSetSessionReasonLabel       = "Reason for change (optional)"
	ManageSetSessionReasonPlaceholder = "e.g. DM unavailable this week"
	ManageSetSessionRescheduleThread  = ":calendar: Session rescheduled to **%s** — _%s_"
	ManageSetSessionRescheduleSuccess = ":white_check_mark: Session for **%s** rescheduled to **%s** — %s. Reason posted to thread."

	// Session reminder DM (sent ~1 hour before NextSession).
	ReminderContent = ":information_source: **Session Reminder: %s**\nYour next session starts in about 1 hour — **%s** (%s)"

	InviteSentMessage      = ":white_check_mark: Invitation sent to <@%s> for **%s**."
	InviteDMMessage        = ":information_source: You've been invited to join **%s** by <@%s>!"
	InviteAcceptedDMUpdate = ":white_check_mark: You accepted the invitation to **%s**."
	InviteDeclinedDMUpdate = ":information_source: You declined the invitation to **%s**."
	InviteAlreadyProcessed = ":information_source: This invitation has already been processed."
	InviteCampaignFull     = ":warning: Cannot invite — campaign **%s** is full."
)

// Session RSVP (buttons on reminder DMs)
const (
	// identifiers
	RSVPAcceptPrefix  = "rsvp_accept"
	RSVPDeclinePrefix = "rsvp_decline"
)
const (
	// user-facing
	RSVPAcceptLabel      = ":white_check_mark: I'm Going!"
	RSVPDeclineLabel     = ":x: I'm Not Going"
	RSVPAcceptedPlayer   = ":white_check_mark: Confirmed! The DM has been notified. May the RNG be with you!"
	RSVPDeclinedPlayer   = ":x: Noted. The DM has been notified. Have a good day!"
	RSVPDMNotifyAccept   = ":green_check_mark: <@%s> confirmed assistance at **%s**\n%s."
	RSVPDMNotifyDecline  = ":x: <@%s> won't be coming for **%s**:\n%s."
	RSVPAlreadyResponded = ":information_source: You've already responded for this session. If you changed your mind, talk to your DM."
	RSVPCampaignGone     = ":information_source: This campaign is no longer active."
)

// Player campaign card (player self-service)
const (
	// identifiers
	PlayerSetSheetPrefix       = "player_set_sheet"
	PlayerSetSheetModalID      = "player_set_sheet_modal"
	PlayerSetSheetFieldID      = "sheet_url"
	PlayerSetTokenPrefix       = "player_set_token"
	PlayerTokenSelectPrefix    = "player_token_select"
	PlayerTokenAssignPrefix    = "player_token_assign"
	PlayerLeaveConfirmPrefix   = "player_leave_confirm"
	PlayerLeaveDoPrefix        = "player_leave_do"
	PlayerContactDMPrefix      = "player_contact_dm"
	PlayerContactDMModalID     = "player_contact_dm_modal"
	PlayerContactDMFieldID     = "dm_message"
	PlayerDownloadTokensPrefix = "player_download_tokens"
	PlayerDownloadSelectPrefix = "player_download_select"
	PlayerDownloadAllValue     = "all"
)
const (
	// user-facing
	PlayerSetSheetModalTitle        = "Set Character Sheet"
	PlayerSetSheetFieldLabel        = "Character Sheet URL"
	PlayerSetSheetFieldPlaceholder  = "https://www.dndbeyond.com/characters/..."
	PlayerSetSheetSuccess           = ":white_check_mark: Character sheet updated for **%s**."
	PlayerSetSheetFailed            = ":no_entry_sign: Failed to save character sheet."
	PlayerSetSheetLabel             = "Set Sheet"
	PlayerOpenSheetLabel            = "Open Sheet"
	PlayerSetTokenLabel             = "Set Token"
	PlayerTokenSelectPlaceholder    = "Select a token..."
	PlayerTokenNewHint              = "Use `/newtoken` to create a new one."
	PlayerTokenAssignLabel          = "Assign to this campaign"
	PlayerTokenAssignSuccess        = ":white_check_mark: Token assigned to **%s**."
	PlayerNoTokens                  = ":information_source: No tokens found. Create one first with `/newtoken`."
	PlayerLeaveConfirmLabel         = "Yes, Leave"
	PlayerLeaveConfirmMsg           = ":warning: Are you sure you want to leave **%s**? This cannot be undone."
	PlayerLeaveCancelLabel          = "Cancel"
	PlayerContactDMModalTitle       = "Send a Message to Your DM"
	PlayerContactDMFieldLabel       = "Message"
	PlayerContactDMFieldPlaceholder = "Ask about scheduling, character questions, lore..."
	PlayerContactDMLabel            = "Contact DM"
	PlayerContactDMSuccess          = ":white_check_mark: Your message has been sent to the DM."
	PlayerContactDMReceived         = "**Message from <@%s>** regarding **%s**:\n\n%s"

	PlayerDownloadTokensLabel       = ":arrow_down: Download Tokens"
	PlayerDownloadSelectPlaceholder = "Select a campaign..."
	PlayerDownloadAllLabel          = ":european_castle: All campaigns"
	PlayerDownloadNoTokens          = ":information_source: You haven't assigned tokens to any campaigns yet. Open a campaign card and use **Set Token** to assign one."
	PlayerDownloadContent           = ":information_source: Here are your %d token(s):"
)

// Timezone preference
const (
	// identifiers
	TimezonePrefix   = "set_timezone"
	TimezoneSelectID = "timezone_select"
)
const (
	// user-facing
	TimezoneLabel             = "Timezone"
	TimezoneHeader            = "**Set your timezone**\nTimes will be shown in your local time.\nCurrent: **%s**"
	TimezoneSelectPlaceholder = "Select your timezone…"
	TimezoneSuccess           = ":white_check_mark: Timezone set to **%s**."
	TimezoneInvalid           = ":warning: Unknown timezone. Please select from the list."
)
