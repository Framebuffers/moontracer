package messages

import (
	"math/rand"
	"strings"
)

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
	BotVersion = "v1.1.1b-adamantine"
	// 2026-05-17: took a long time to get here, but here we are. v1.0. time to roll initiative for the first release, i guess
	// 2026-05-29: started implementing for real on my first big server
)
const (
	// user-facing
	GenericErrorMessage      = "🚫 Something went wrong."
	InvalidButtonDataMessage = "⚠️ Invalid button data."
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
	DateInputFormat     = "02/01/2006"       // DD/MM/YYYY
	TimeInputFormat     = "15:04"            // HH:MM (24h)
	DateTimeInputFormat = "02/01/2006 15:04" // combined for ParseInLocation
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
	NotRegisteredMessage       = "⚠️ You need to register first."
	AlreadyRegisteredMessage   = "ℹ️ You are already registered!"
	RegistrationFailureMessage = "🚫 Failed to register. Please try again later."
	RegistrationSuccessMessage = "🐺 Welcome, <@%s>! You are now registered."
)

// Campaign lookup
const (
	// internal log prefixes
	CampaignFetchError      = "campaign: error fetching campaign %s: "
	PlayerFetchErrorMessage = "models.GetCampaignPlayers(): Error fetching players: "
)
const (
	// user-facing
	CampaignNotFoundMessage         = "ℹ️ Campaign not found."
	CampaignArchivedFooter          = "ℹ️ This campaign is archived. It is read-only now."
	CampaignLoadFailureErrorMessage = "🚫 Failed to load campaign."
	CampaignPlayersLoadError        = "🚫 Failed to load campaign players."
)

// Campaign creation
const (
	// internal log prefix
	CampaignCreationFailureErrorMessage = "campaign.CreateCampaign(): error creating campaign: "
)
const (
	// user-facing
	SlotCountMismatchErrorMessage              = "🚫 Invalid slot count. Capacity must be a positive number. Leave the field empty for unlimited."
	CampaignAndRegistrationFailureErrorMessage = "🚫 Failed to create campaign. Make sure you are registered."
	CampaignCreationMessage                    = "✅ You just created a new campaign: "
	CampaignStaffNotifyFailureMessage          = "⚠️ Could not notify staff members to ask for approval of this Campaign."
	CampaignApprovalRequestMessage             = "ℹ️ New campaign **%s** by <@%s> needs approval."
)

// Campaign join
const (
	// internal log prefix
	InsertPlayerErrorMessage = "db.Insert(): Error inserting Campaign Player: "
)
const (
	// user-facing
	CampaignClosedMessage          = "⚠️ This campaign is not open for new players."
	PlayerBannedMessage            = "⛔ You are banned from this campaign."
	PlayerAlreadyOnCampaignMessage = "ℹ️ You are already in this campaign."
	CampaignFullMessage            = "⚠️ This campaign is full."
	PlayerFailedToJoinMessage      = "🚫 Failed to join campaign."
	PlayerJoinedCampaignMessage    = "🐺 You have joined **%s**!"

	// Westmarch session-capacity tripwire (FCFS soft alert).
	WestmarchOverCapacityDMAlert      = "⚠️ **INFO:** <@%s> just joined westmarch **%s**. party is now %d active player(s). Session capacity is %d. They've been admitted; bring an extra seat or trim attendance for the next session."
	WestmarchOverCapacityPlayerNotice = "🐺 You're in **%s**! \n⚠️ Warning!: this westmarch's session capacity (%d) is already met, so the DM has been notified. Talk to your DM for more help."
)

// Campaign leave
const (
	// internal log prefix
	LeavingCampaignErrorMessage = "models.RemoveCampaignPlayer(): error removing player: "
)
const (
	// user-facing
	MasterIsLeavingCampaignErrorMessage = "⚠️ You are the DM. You cannot leave your own campaign."
	FailedToLeaveCampaignErrorMessage   = "🚫 Failed to leave campaign."
	PlayerLeftCampaignMessage           = "ℹ️ You have left **%s**."
)

// Campaign toggle
const (
	// internal log prefix
	CampaignUpdateErrorMessage = "db.Update(): error updating campaign: "
)
const (
	// user-facing
	MasterCanToggleStatusErrorMessage = "ℹ️ Only the DM can toggle campaign status."
	CampaignStatusMessage             = "ℹ️ **%s** is now **%s**."
)

// My campaigns (all user-facing)
const (
	NoCampaignsMessage   = "ℹ️ You are not in any campaigns yet!"
	MyCampaignsLoadError = "⚠️ Failed to load your campaigns."
)

// Campaign embed UI labels
const (
	// identifiers
	EmbedColor = 0x5865F2
)
const (
	// user-facing
	OpenCampaignLabel          = "🔵 Set as Open Campaign"
	ClosedCampaignLabel        = "🔴 Set as Closed Campaign"
	LeaveCampaignLabel         = "🚪 Leave Campaign"
	JoinCampaignLabel          = "🟢 Join Campaign"
	ClosedStatusLabel          = "🚫 Closed"
	OpenStatusLabel            = "🟢 Open"
	ArchivedStatusLabel        = "🟡 Archived"
	CampaignLabel              = "🏰 Campaign"
	CampaignTypeOneShotLabel   = "📖 One-shot"
	CampaignTypeWestmarchLabel = "🛡️ Westmarch"
	NoneLabel                  = "None"
	NoBooksSpecifiedLabel      = "⚠️ No books specified"
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
	NewCampaignModalTitle  = "🏰 Create a New Campaign"
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
	AddPlayerNotDMOrModMessage   = "ℹ️ You must be the DM of this campaign to add players."
	AddPlayerTargetNotRegistered = "ℹ️ That user is not registered. They need to `/register` first."
	AddPlayerAlreadyInCampaign   = "⚠️ That player is already in this campaign."
	AddPlayerCampaignFullMessage = "🚫 This campaign is full."
	AddPlayerFailureMessage      = "⚠️ Failed to add player to campaign."
	AddPlayerSuccessMessage      = "🐺 Added <@%s> to **%s**!"
)

// Ban
const (
	// identifiers
	BanCommandName = "ban"
)
const (
	// user-facing
	BanCommandDesc          = "Globally ban a player from the server."
	BanCannotBanSelf        = "⚠️ You cannot ban yourself."
	BanInsufficientRole     = "⚠️ You cannot ban someone of equal or higher role."
	BanTargetNotFound       = "ℹ️ That player is not registered."
	BanPlayerNotInCampaign  = "⚠️ The player ID is not valid for this campaign, or it does not belong to it."
	BanTargetAlreadyBanned  = "ℹ️ That player is already banned."
	BanFailureMessage       = "🚫 Failed to ban player."
	BanSuccessMessage       = "ℹ️ Banned <@%s>. \nThey can no longer interact with the bot."
	BanSuccessReasonMessage = "ℹ️ Banned <@%s>: %s"
)

// Unban
const (
	// identifiers
	UnbanCommandName = "unban"
)
const (
	// user-facing
	UnbanCommandDesc     = "Lift a global ban from a player."
	UnbanTargetNotFound  = "ℹ️ That player is not registered."
	UnbanTargetNotBanned = "ℹ️ That player is not banned."
	UnbanFailureMessage  = "🚫 Failed to unban player."
	UnbanSuccessMessage  = "ℹ️ Unbanned <@%s>. They can interact with the bot again."
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
	CampaignArchivedMessage = "ℹ️ This campaign has been archived and can no longer be modified."
	AbandonCommandDesc      = "Archive your campaign permanently. Only the DM can do this."
	AbandonNotDMMessage     = "ℹ️ Only the DM of this campaign can abandon it."
	AbandonFailureMessage   = "🚫 Failed to archive campaign."
	AbandonSuccessMessage   = "ℹ️ Campaign **%s** has been archived. It is now an immutable record, and cannot be recovered."
)

// Manage campaigns
const (
	// identifiers
	ManageCampaignsCommandName = "managecampaigns"
	ManageCommandName          = "manage"
	ManageCommandOptionName    = "campaign"
)
const (
	// user-facing
	ManageCampaignsCommandDesc = "Manage campaigns you run as DM."
	ManageCommandDesc          = "Manage one of your campaigns."
	ManageCommandOptionDesc    = "Campaign to manage."
	ManageCampaignsLabel       = "Manage Campaigns"
	ManageNoDMCampaigns        = "⚠️ You are not the DM of any campaigns."
	ManageNotAuthorized        = "ℹ️ You must be the DM of this campaign to manage it."
	ManageCampaignNotFound     = "ℹ️ Campaign not found."
	ManageDeleteInProgress     = "⏳ Deleting campaign..."
	ManageDeleteSuccess        = "ℹ️ Campaign **%s** has been deleted."
	ManageDeleteFailure        = "🚫 Failed to delete campaign."
	ManageBanNoMembers         = "ℹ️ This campaign has no members to ban."
	ManageCampaignBanSuccess   = "ℹ️ Banned <@%s> from **%s**."
	ManageCampaignBanFailure   = "🚫 Failed to ban player from campaign."
	ManageSelectMember         = "Select a member to ban from **%s**:"
	ManageBanSelectPlaceholder = "Select a member..."
	ManageCampaignHeader       = "Managing **%s**:"
	ManageInviteSelectPrompt   = "Select a player to invite to **%s**:"
	MyCampaignListLine         = "**%s**- %s (%s)"
	ManageCampaignListLine     = "**%s**- %s"
)

// Token upload
const (
	// identifiers- token gallery
	TokensCommandName              = "tokens"
	TokenGallerySelectPrefix       = "token_gallery_select"
	TokenGalleryAssignPrefix       = "token_gallery_assign"
	TokenGalleryAssignSelectPrefix = "token_gallery_assign_select"
	TokenDeletePromptPrefix        = "token_delete_prompt"
	TokenDeleteConfirmPrefix       = "token_delete_confirm"
	TokenDownloadPrefix            = "token_download"
)
const (
	// user-facing- token gallery
	TokensCommandDesc                   = "Browse and manage your tokens."
	TokensLabel                         = "🐺 Tokens"
	TokenGalleryHeader                  = "Your tokens:"
	TokenGalleryNone                    = "ℹ️ You have no tokens yet! Use `/newtoken` to create one."
	TokenNotAssigned                    = "ℹ️ This token is not assigned to any campaign"
	TokenGallerySelectPlaceholder       = "Select a token to manage..."
	TokenDownloadLabel                  = "⬇️ Download"
	TokenDeleteLabel                    = "🗑️ Delete"
	TokenDeleteConfirmMsg               = "⚠️ Are you sure you want to delete **%s**? This cannot be undone."
	TokenDeleteConfirmLabel             = "🗑️ Yes, Delete"
	TokenDeleteCancelLabel              = "❌ Cancel"
	TokenDeleteSuccess                  = "ℹ️ Token **%s** deleted."
	TokenDeleteFailed                   = "⚠️ Failed to delete token."
	TokenGalleryAssignPrompt            = "ℹ️ Assign **%s** to a campaign:"
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
	TokenUploadNeedOneOf             = "🚫 Provide either a frame image or a color- not both, not neither."
	TokenUploadNotImage              = "⚠️ Both files must be images (JPEG or PNG)."
	TokenUploadTooLarge              = "⚠️ Each file must be under 8 MiB."
	TokenUploadProcessFailed         = "🚫 Failed to process your token. Please try again."
	TokenUploadPreviewContent        = "ℹ️ Here's a preview of your token. Apply it to your profile or discard."
	TokenApplyLabel                  = "✅ Apply"
	TokenDiscardLabel                = "❌ Discard"
	TokenPostcreateSelectPlaceholder = "Assign to a campaign..."
	TokenSkipLabel                   = "⬅️ Skip for now"
	TokenPostcreateHeader            = "✅ **%s** saved! Assign it to one of your campaigns, or skip."
	TokenPostcreateAssigned          = "🐺 Token assigned to **%s**!"
	TokenSavedNoAssign               = "ℹ️ Token saved. You can assign it later from your campaign card."
	TokenNameModalTitle              = "Name Your Token"
	TokenNameFieldLabel              = "Character Name"
	TokenNameFieldPlaceholder        = "e.g. Soft Doggo"
	TokenApplySuccess                = "✅ Token saved as **%s**!"
	TokenDiscardSuccess              = "ℹ️ Token discarded."
	TokenApplyFailed                 = "⚠️ Failed to save token. Please try again."
)

// Campaign cover / upload
const (
	// identifiers
	CampaignUploadCommandName     = "uploadcover"
	CampaignUploadKindOptName     = "kind"
	CampaignUploadKindCoverChoice = "Cover"
	CampaignUploadCampaignOptName = "campaign"
	CampaignUploadImageOptName    = "image"
)
const (
	// user-facing
	CampaignUploadCommandDesc     = "Upload a cover image for one of your campaigns."
	CampaignUploadKindOptDesc     = "What kind of image to upload."
	CampaignUploadCampaignOptDesc = "The campaign to upload an image for."
	CampaignUploadImageOptDesc    = "Image file (JPEG/PNG/WebP, up to 8 MiB)."
	CampaignUploadNotDM           = "ℹ️ Only the DM of this campaign can change its cover."
	CampaignUploadNotImage        = "⚠️ That file doesn't look like an image. Try JPEG, PNG, or WebP."
	CampaignUploadTooLarge        = "⚠️ Image is too large. Max 8 MiB."
	CampaignUploadMissingAttach   = "ℹ️ No image attached. Attach a file to the `image` option."
	CampaignUploadFailure         = "⚠️ Failed to save cover. Please try again."
	CampaignUploadSuccess         = "✅ Cover set for **%s**. [View](%s)"
	SetCoverButtonLabel           = "Set Cover"
	SetCoverInstructions          = "Use `/uploadcover campaign:<name> image:<file>` to set a cover for this campaign."
)

// Manage campaigns: button labels (all user-facing)
const (
	ManageDeleteLabel           = "🗑️ Delete"
	ManageBanLabel              = "🚫 Ban Member"
	ManageAnnounceLabel         = "🏰 New Session"
	ManageRescheduleLabel       = "📅 Configure Schedule"
	ManageCampaignButtonLabel   = "⚙️ Manage"
	ManageDownloadTokensLabel   = "⬇️ Player Tokens"
	ManageDownloadTokensNone    = "ℹ️ No players in this campaign have tokens assigned."
	ManageDownloadTokensContent = "ℹ️ Tokens for **%s** (%d player(s)):"
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
	SetRoleNotDMOrMod          = "⚠️ You must be the DM of this campaign to set its role."
	SetRoleSuccess             = "✅ Linked role **%s** to campaign **%s**."
	SetRoleCreateFailed        = "⚠️ Failed to create Discord role."
	SetRoleUpdateFailed        = "⚠️ Failed to update campaign."
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
	ApproveButtonLabel            = "✅ Approve"
	DenyButtonLabel               = "🚫 Deny"
	CampaignApprovedMessage       = "🐺 Campaign **%s** has been approved."
	CampaignDeniedMessage         = "🚫 Campaign **%s** has been denied and deleted."
	CampaignApproveNotModError    = "⚠️ Only mods or admins can approve campaigns."
	CampaignApproveNotFound       = "ℹ️ Campaign not found or already processed."
	CampaignApproveError          = "⚠️ Failed to process campaign approval."
	CampaignDenyModalTitle        = "🚫 Deny Campaign"
	CampaignDenyReasonLabel       = "Reason"
	CampaignDenyReasonPlaceholder = "Why is this campaign being denied?"
	CampaignDeniedDMMessage       = "🚫 Your campaign **%s** has been denied. Reason: %s"
	CampaignApprovedDMMessage     = "🐺 Your campaign **%s** has been approved!"
	CampaignApprovedStatusMessage = "✅ **%s** approved. Channels set up and DM notified."
	CampaignDeniedStatusMessage   = "🚫 **%s** denied and removed. DM notified with reason: _%s_"
	CampaignDenyPendingMessage    = "ℹ️ Campaign **%s**: denial in progress..."
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
	AnnounceFieldLabel       = "✉️ Message"
	AnnounceFieldPlaceholder = "Type your announcement to all campaign members..."
	AnnounceSentMessage      = "🗣️ Announcement sent to %d members of **%s**."
	AnnounceNoMembers        = "ℹ️ This campaign has no members to announce to."
	AnnounceError            = "⚠️ Failed to send announcement."
	AnnouncePostedToThread   = "🗣️ Announcement posted to the **%s** announcements thread."
	AnnounceCooldown         = "⏳ You can post another announcement in %s."
	AnnounceThreadContent    = "%s**🗣️Announcement from <@%s>:**\n\n%s"
	AnnounceDMContent        = "**[%s]** 🗣️ Announcement from <@%s>:\n\n%s"
)

// CampaignsCategoryName is the shared Discord category all campaign channels are grouped under.
const CampaignsCategoryName = "Campaigns"

// Thread initial pinned messages, keyed by standard thread name.
const (
	ThreadInitMsgAnnouncements = "📣 Campaign announcements and session news will be posted here."
	ThreadInitMsgSessions      = "📅 Session schedules and notes will be posted here."
	ThreadInitMsgDiceRolls     = "🎲 Roll your dice and share your results here!"

	// ThreadInitMsgWelcomeFmt takes the campaign name as its argument.
	ThreadInitMsgWelcomeFmt    = "🐺 Welcome to **%s**! This is your campaign channel. \nCheck the other threads for announcements, sessions, dice rolls, and general discussion."
	WelcomeThreadCoverReminder = "-# 📷 DM: Don't forget to add a cover image in campaign settings: it'll appear here and in the billboard! Use `/campaignupload` to upload an image."

	/*
		ResourcesThreadSyncFmt is the content posted/edited in the resources thread when links change.

		Args: VTT section (may be empty), links section (may be empty).
	*/
	ResourcesThreadSyncFmt = "## 🔗 Campaign Resources\n\n%s%s"
	ResourcesThreadVTTFmt  = "🎲 **VTT:** %s\n"
	ResourcesThreadLinkFmt = "• %s\n"
	ResourcesThreadEmpty   = "_No resources set yet. The DM can add links in `/manage` → Settings → 🔗 Links._"
)

// ThreadNavEmoji maps thread slot names to their display emoji in the welcome nav section.
var ThreadNavEmoji = map[string]string{
	"announcements": "📣",
	"sessions":      "📅",
	"dice-rolls":    "🎲",
	"characters":    "🧙",
	"memes":         "😂",
	"art":           "🎨",
	"downtime":      "⏳",
	"resources":     "🔗",
}

// ThreadNavOrder defines the display order of threads in the welcome nav section.
var ThreadNavOrder = []string{"announcements", "sessions", "dice-rolls", "characters", "memes", "art", "downtime", "resources"}

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
	RescheduleSuccess          = "✅ Schedule updated for **%s**: %s %s UTC (%sh), %s."
	RescheduleInvalidDay       = "⚠️ Invalid day. Use a name like Monday, Tuesday... or abbreviation like mon, tue..."
	RescheduleInvalidTime      = "⚠️ Invalid time format. Use HH:MM (e.g. 19:00)."
	RescheduleInvalidDuration  = "⚠️ Invalid duration. Enter a number (e.g. 3)."
	RescheduleInvalidFrequency = "⚠️ Invalid frequency. Use: weekly, biweekly, monthly, quarterly, yearly."
	RescheduleError            = "🚫 Failed to update schedule."
)

// Campaign database (debug)
const (
	// identifiers
	CampaignDBCommandName = "campaigndatabase"
)
const (
	// user-facing
	CampaignDBCommandDesc = "Show all campaigns in the database (staff only)."
	CampaignDBEmpty       = "ℹ️ No campaigns in the database."
	CampaignDBNotStaff    = "ℹ️ Only mods or admins can use this command."
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
	MeHubMessage      = "🐺 Hey, <@%s>! What would you like to do?"
	MeCampaignsLabel  = "🏰 Campaigns"
	MeConfigLabel     = "⚙️ Configuration"
	MeCampaignsHeader = "🏰 Campaigns"
	MeConfigHeader    = "⚙️ Configuration"
	ControlPanelLabel = "🎛️ Control Panel"
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
	CampaignsNoneAvailable     = "ℹ️ No campaigns match this filter."
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
	SearchNoResults   = "ℹ️ No campaigns found matching that name."
)

// Admin hub (/admin)
const (
	// identifiers
	AdminCommandName = "admin"
)
const (
	// user-facing
	AdminCommandDesc     = "Mod/Admin panel."
	AdminNotStaff        = "ℹ️ Only mods or admins can use this command."
	DebugSurfaceDisabled = "ℹ️ This surface is disabled in production. Set DEV_MODE=true to re-enable."
	AdminHubMessage      = "Admin Panel:"
	AdminCampaignsLabel  = "ℹ️ Query Campaigns"
	AdminDMsLabel        = "✉️ DMs"
	AdminBroadcastLabel  = "🗣️ Broadcast"
	AdminDatabaseLabel   = "🗃️ Database"
	AdminSettingsLabel   = "⚙️ Settings"
	AdminDiagLabel       = "🐺 Diagnostics"
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
	AboutCommandGitHubLabel  = "💻 GitHub"
	AboutLabel               = "🐺 About"
	HelpLabel                = "❓ Help"
	AboutCommandBotDesc      = "...a D&D campaign manager for players, DM and spectators!"
	AboutCommandCopyright    = "(C) 2026 **Framebuffer**"
	AboutCommandHelp         = "Type `/help`, or press the **Help** button below for a list of commands."
	AboutCommandAwoo         = "awoo!"
	AboutCommandAttributions = "Thanks to the D&D r/Chile Discord server for giving me the idea, letting me test the bot on their server and give me feedback to improve this bot."
)

// Navigation buttons (all user-facing)
const (
	BackLabel = ""
	HomeLabel = "🏠"
)

// Hub button labels (all user-facing)
const (
	MyCampaignsLabel     = "🏰 My Campaigns"
	NextSessionsLabel    = "📅 Next Sessions"
	NotificationsLabel   = "🔔 Alerts"
	BrowseCampaignsLabel = "📚 Browse Campaigns"
	MyProfileLabel       = HomeLabel
	NewCampaignLabel     = "📖 New Campaign"
	AdminPanelLabel      = "⚙️ Admin Panel"
)

// Manage campaign hub labels (all user-facing)
const (
	ManagePlayersLabel   = "🐺 Players"
	ManageSessionsLabel  = "📑 Sessions"
	ManageSettingsLabel  = "⚙️ Settings"
	ManageDangerLabel    = "⚠️ Spicy Zone"
	ManageOpenLabel      = "🟢 Open Campaign"
	ManageCloseLabel     = "🔴 Close Campaign"
	CampaignAutoClosedDM = "ℹ️ **%s** has been automatically closed- all %d slots are filled."
)

// Manage campaign: Links
const (
	// identifiers
	ManageLinksPrefix  = "manage_links"
	ManageLinksModalID = "modal_manage_links"

	ManageGameInfoPrefix  = "manage_game_info"
	ManageGameInfoModalID = "modal_manage_game_info"
)
const (
	// user-facing
	ManageLinksLabel                = "🔗 Links"
	ManageLinksModalTitle           = "🔗 Campaign Links"
	ManageLinksVTTLabel             = "🔗 VTT Link"
	ManageLinksVTTPlaceholder       = "https://owlbear.rodeo/... or https://app.roll20.net/..."
	ManageLinksResourcesLabel       = "Session Resources (one URL per line)"
	ManageLinksResourcesPlaceholder = "https://drive.google.com/...\nhttps://example.com/map"
	ManageLinksSuccess              = "✅ Links updated for **%s**."
	ManageLinksEmbedTitle           = "🔗 Links"

	ManageGameInfoLabel            = "🎲 Game Info"
	ManageGameInfoModalTitle       = "🎲 Game Details"
	ManageGameInfoRulesLabel       = "House rules / variant (optional)"
	ManageGameInfoRulesPlaceholder = "e.g. Milestone levelling, no flanking..."
	ManageGameInfoVTTLabel         = "VTT platform (optional)"
	ManageGameInfoVTTPlaceholder   = "e.g. Roll20, Foundry VTT, Owlbear Rodeo..."
	ManageGameInfoBooksLabel       = "Books allowed (comma-separated, optional)"
	ManageGameInfoBooksPlaceholder = "e.g. PHB, Xanathar's, Tasha's"
	ManageGameInfoExtraLabel       = "Extra info for players (optional)"
	ManageGameInfoExtraPlaceholder = "Tone, session zero notes, Discord links..."
	ManageGameInfoSuccess          = "✅ Game info updated for **%s**."
	ManageReminderLinks            = "\n\n🔗 **Links for tonight:**"
	ManageReminderVTT              = "\nVTT: %s"
	ManageReminderSheets           = "\nSheets: %s"
	ManageReminderResource         = "\n• %s"
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
	ManageSetRoleLabel = "👥 Set Role"
	ManageArchiveLabel = "📂 Archive"

	// Set Role modal
	ManageSetRoleModalTitle = "🔗 Link Discord Role"
	ManageSetRoleFieldLabel = "Role name (creates if it doesn't exist)"
	ManageSetRoleSuccess    = "✅ Linked role **%s** to campaign **%s**."
	ManageSetRoleFailed     = "🚫 Failed to set role."

	// Delete confirmation + handler
	ManageDeleteConfirm      = "⚠️ Are you sure you want to delete **%s**? This is permanent and cannot be undone. All members will be removed."
	ManageDeleteConfirmLabel = "✅ Yes, Delete"
	ManageDeleteCancelLabel  = "❌ Cancel"

	// Archive confirmation + handler
	ManageArchiveConfirm      = "⚠️ Are you sure you want to archive **%s**? This is permanent and cannot be undone."
	ManageArchiveConfirmLabel = "✅ Yes, Archive"
	ManageArchiveCancelLabel  = "❌ Cancel"
	ManageArchiveInProgress   = "⏳ Archiving campaign..."
	ManageArchiveSuccess      = "ℹ️ Campaign **%s** has been archived. It is now an immutable record."
	ManageArchiveFailed       = "🚫 Failed to archive campaign."
)

// New campaign config (post-modal dropdowns)
const (
	// identifiers
	NewCampaignBookPrefix            = "newcampaign_book"
	NewCampaignFormatPrefix          = "newcampaign_format"
	NewCampaignFreqPrefix            = "newcampaign_freq"
	NewCampaignSubmitPrefix          = "newcampaign_submit"
	NewCampaignCancelPrefix          = "newcampaign_cancel"
	NewCampaignScheduleModalID       = "modal_newcampaign_schedule"
	NewCampaignGameDetailsOpenPrefix = "newcampaign_gamedetails_open"
	NewCampaignSubmitApprovalPrefix  = "newcampaign_submit_approval"

	NewCampaignScheduleDateFieldID = "newcampaign_sched_date"
	NewCampaignScheduleTimeFieldID = "newcampaign_sched_time"
)
const (
	// user-facing
	NewCampaignBookPlaceholder      = "Select a game system..."
	NewCampaignFormatPlaceholder    = "Select a format..."
	NewCampaignFreqPlaceholder      = "Select a schedule frequency..."
	NewCampaignConfigMessage        = "ℹ️ Campaign **%s** created (pending setup).\n\nSelect a game system, format, and frequency, then submit for approval. You'll set the first session date in the next step."
	NewCampaignSubmitLabel          = "✅ Submit for Approval"
	NewCampaignCancelLabel          = "❌ Cancel"
	NewCampaignSubmittedMessage     = "✅ Campaign **%s** has been submitted for approval!"
	NewCampaignCancelledMessage     = "ℹ️ Campaign creation cancelled."
	NewCampaignMissingConfigMessage = "⚠️ Please select a game system and format before submitting."

	NewCampaignScheduleModalTitle      = "📅 When's your first session?"
	NewCampaignScheduleDateLabel       = "Date (DD/MM/YYYY)"
	NewCampaignScheduleTimeLabelFmt    = "Time (%s)"
	NewCampaignScheduleDatePlaceholder = "e.g. 14/06/2026"
	NewCampaignScheduleTimePlaceholder = "e.g. 19:00"
	NewCampaignScheduleSkipHint        = "Leave blank to set a session later from campaign settings."
	NewCampaignScheduleInvalidDate     = "⚠️ Invalid date format. Use DD/MM/YYYY (e.g. 14/06/2026)."
	NewCampaignScheduleInvalidTime     = "⚠️ Invalid time format. Use HH:MM (e.g. 19:00)."
	NewCampaignScheduleInPast          = "⚠️ That date/time is in the past. Please pick a future date."

	NewCampaignGameDetailsPrompt    = "📅 Schedule saved! **Last step:** add game details (rules, VTT, books, extra info)... or skip and submit for approval now."
	NewCampaignGameDetailsOpenLabel = "🎲 Add Game Details"
	NewCampaignSubmitApprovalLabel  = "✅ Submit for Approval"

	// Book dropdown option labels
	NewCampaignBookLabel5e    = "D&D 5e"
	NewCampaignBookLabel55e   = "D&D 5.5e (2024)"
	NewCampaignBookLabelPF2e  = "Pathfinder 2e"
	NewCampaignBookLabelOther = "Other / Homebrew"

	// Frequency dropdown option labels
	NewCampaignFreqLabelWeekly   = "Weekly"
	NewCampaignFreqLabelBiweekly = "Bi-weekly"
	NewCampaignFreqLabelMonthly  = "Monthly"
	NewCampaignFreqLabelOnce     = "Once"

	// Browse more after joining
	BrowseMoreLabel = "🔍 Browse more campaigns"
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
	MyCampaignsListHeader      = "🏰 Your campaigns:\n"
	ManageCampaignsListHeader  = "🏰 Your campaigns (DM):\n"
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
	NextSessionsPrefix      = "next_sessions"
	NextSessionsCommandName = "nextsessions"
	ThisWeekCommandName     = "thisweek"
)
const (
	// user-facing
	NextSessionsHeader      = "Upcoming sessions:"
	NextSessionsNone        = "ℹ️ You have no upcoming sessions."
	NextSessionsCommandDesc = "Show your upcoming sessions."

	ThisWeekHeader      = "📅 Sessions happening this week:"
	ThisWeekNone        = "ℹ️ No sessions are scheduled in the next 7 days."
	ThisWeekCommandDesc = "Browse all sessions scheduled this week."
	ThisWeekSlotsFmt    = "%d/%d slots"
	ThisWeekSlotsOpen   = "open slots"
	ThisWeekFull        = "full"
	ThisWeekClosed      = "closed"
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
	NotificationsHeader     = "🔔 Alert settings:"
	NotificationsNone       = "ℹ️ No notification preferences configured yet."
	NotifLabelAnnouncements = "🗣️ Announcements"
	NotifLabelSessions      = "📅 Session Reminders"
	NotifLabelInvitations   = "✉️ Invitations"
	NotifLoadFailed         = "🚫 Failed to load notification settings."
	NotifUpdateFailed       = "🚫 Failed to update notification settings."
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
	AdminCampaignsNone             = "ℹ️ No campaigns in the database."
	AdminCampaignSelectPlaceholder = "Pick a campaign for details..."
	AdminContactDMLabel            = "📬 Contact DM"
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
	AdminBroadcastModalTitle = "📢 Broadcast Message"
	AdminBroadcastFieldLabel = "✉️ Message"
	AdminBroadcastSuccess    = "✅ Broadcast sent."
	AdminBroadcastSent       = "✅ Broadcast sent to %d players."
	AdminBroadcastFailed     = "🚫 Failed to send broadcast."
	AdminBroadcastDMContent  = "**🗣️ Broadcast from <@%s>:**\n\n%s\n\n-# _You can disable these in `/me` -> Configuration -> 🔔 Alerts._"
)

// Admin hub: Database viewer
const (
	// identifiers
	AdminDatabasePrefix = "admin_database"
)
const (
	// user-facing
	AdminDBCampaignLine = "**%s** (`%s`)- DM: <@%s> [%s]"
)

// Admin hub: Settings
const (
	// identifiers
	AdminSettingsPrefix             = "admin_settings"
	AdminBillboardSetPrefix         = "admin_billboard_set"
	AdminBillboardFormatCampaign    = "campaign"
	AdminBillboardFormatOneshot     = "oneshot"
	AdminBillboardFormatWestmarch   = "westmarch"
	AdminBillboardSetCategoryPrefix = "admin_billboard_set_category"
	AdminCampaignChannelSetPrefix   = "admin_campaign_channel_set"
	AdminCampaignsCategorySetPrefix = "admin_campaigns_category_set"
	AdminArchivedCategorySetPrefix  = "admin_archived_category_set"
)
const (
	// user-facing
	AdminSettingsHeader                = "Bot settings:"
	AdminSettingsGeneralHeader         = "**⚙️ General Settings**\n\n"
	AdminBillboardHeader               = "**⚙️ Billboard Channels**\nSelect which forum channel each campaign format posts to when approved.\n\n"
	AdminBillboardChannelsLabel        = "Billboard channels ➡️"
	AdminBillboardCampaignLabel        = "🏰 Campaigns"
	AdminBillboardOneshotLabel         = "📖 One-shots"
	AdminBillboardWestmarchLabel       = "🛡️ Westmarches"
	AdminBillboardCampaignPlaceholder  = "Select forum channel for campaigns…"
	AdminBillboardOneshotPlaceholder   = "Select forum channel for one-shots…"
	AdminBillboardWestmarchPlaceholder = "Select forum channel for westmarches…"
	AdminBillboardSavedFmt             = "✅ Billboard channel for **%s** set."
	AdminBillboardCurrentFmt           = "Current: %s"
	AdminBillboardNotSet               = "_(not set -auto-create)_"

	AdminBillboardCategoryLabel       = "📁 Billboard category"
	AdminBillboardCategoryPlaceholder = "Select category for billboard channels…"
	AdminBillboardCategorySavedFmt    = "✅ Billboard category set to %s."

	AdminCampaignsCategoryLabel       = "📁 Campaign channels category"
	AdminCampaignsCategoryPlaceholder = "Select category for new campaign channels…"

	AdminArchivedCategoryLabel       = "🗄️ Archived campaigns category"
	AdminArchivedCategoryPlaceholder = "Select category for retired campaign channels…"

	AdminCampaignChannelLabel       = "📢 Campaign channel"
	AdminCampaignChannelPlaceholder = "Select campaign announcements channel…"
	AdminCampaignChannelSavedFmt    = "✅ Campaign channel set to %s."

	AdminAuditLogChannelLabel       = "📋 Audit Log Channel"
	AdminAuditLogChannelPlaceholder = "Select staff-only audit log channel…"
	AdminAuditLogChannelSetPrefix   = "admin_audit_log_channel_set"
)

// Admin hub: Nuke
const (
	AdminNukeLabel             = "💣 Nuke Campaign"
	AdminNukePrefix            = "admin_nuke"
	AdminNukeSelectPrefix      = "admin_nuke_select"
	AdminNukeConfirmPrefix     = "admin_nuke_confirm"
	AdminNukeCancelPrefix      = "admin_nuke_cancel"
	AdminNukeSelectPlaceholder = "Pick a campaign to permanently delete…"
	AdminNukeConfirmFmt        = "⚠️ **CONFIRM NUKE** ⚠️\n\nYou are about to permanently delete **%s** (`%s`) from the database *and* Discord.\n\n**This cannot be undone.** All players, sessions, and session data will be erased. An audit entry will be kept."
	AdminNukeInProgress        = "💣 Nuking campaign…"
	AdminNukeCancelled         = "❌ Nuke cancelled."
	AdminNukeChannelMismatch   = "🚫 Safety check failed: the Discord channel does not match the DB record. Aborting."
	AdminNukeSuccess           = "✅ Campaign **%s** permanently deleted."
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
	ManageInviteLabel               = "👤 Invite Player"
	ManageSetSessionLabel           = "📅 Set Session"
	ManageRescheduleSessionLabel    = "⌚ Reschedule"
	ManageSetSessionModalTitle      = "🏰 Set Next Session"
	ManageSetSessionDateLabel       = "Date (DD/MM/YYYY)"
	ManageSetSessionDatePlaceholder = "08/05/2026"
	ManageSetSessionTimeLabel       = "Time UTC (HH:MM, 24h)"
	ManageSetSessionTimePlaceholder = "19:00"
	ManageSetSessionInvalidDate     = "⚠️ Invalid date format. Use DD/MM/YYYY."
	ManageSetSessionInvalidTime     = "⚠️ Invalid time format. Use HH:MM (24h)."
	ManageSetSessionInPast          = "⚠️ Cannot set a session in the past."
	ManageSetSessionSuccess         = "✅ Next session for **%s** set to **%s**- %s."
	ManageSetSessionUpdateFailed    = "🚫 Failed to update next session."

	// Reschedule-specific (existing session -> new date + reason).
	ManageRescheduleModalTitle        = "Reschedule Session"
	ManageSetSessionReasonLabel       = "Reason for change (optional)"
	ManageSetSessionReasonPlaceholder = "e.g. DM unavailable this week"
	ManageSetSessionRescheduleThread  = "📅 Session rescheduled to **%s**- _%s_"
	ManageSetSessionRescheduleSuccess = "✅ Session for **%s** rescheduled to **%s**- %s. Reason posted to thread."

	// Session reminder DM (sent ~1 hour before NextSession).
	ReminderContent = "ℹ️ **Session Reminder: %s**\nYour next session starts in about 1 hour- **%s** (%s)"

	InviteSentMessage      = "✅ Invitation sent to <@%s> for **%s**."
	InviteDMMessage        = "ℹ️ You've been invited to join **%s** by <@%s>!"
	InviteAcceptedDMUpdate = "✅ You accepted the invitation to **%s**."
	InviteDeclinedDMUpdate = "ℹ️ You declined the invitation to **%s**."
	InviteAlreadyProcessed = "ℹ️ This invitation has already been processed."
	InviteCampaignFull     = "⚠️ Cannot invite- campaign **%s** is full."
)

// Session response - legacy (campaign-level, reminder DMs only)
const (
	// identifiers
	ResponseAcceptPrefix  = "response_accept"
	ResponseDeclinePrefix = "response_decline"
)
const (
	// user-facing
	ResponseAcceptLabel      = "✅ I'm Going!"
	ResponseDeclineLabel     = "❌ I'm Not Going"
	ResponseAcceptedPlayer   = "✅ Confirmed! The DM has been notified. May the RNG be with you!"
	ResponseDeclinedPlayer   = "❌ Noted. The DM has been notified. Have a good day!"
	ResponseDMNotifyAccept   = "✅ <@%s> confirmed assistance at **%s**\n%s."
	ResponseDMNotifyDecline  = "❌ <@%s> won't be coming for **%s**:\n%s."
	ResponseAlreadyResponded = "ℹ️ You've already responded for this session. If you changed your mind, talk to your DM."
	ResponseCampaignGone     = "ℹ️ This campaign is no longer active."
)

// New Session command + per-session response (sessions table)
const (
	// identifiers
	NewSessionCommandName  = "newsession"
	NewSessionOptionName   = "campaign"
	NewSessionModalID      = "modal_new_session"
	NewSessionDateFieldID  = "new_session_date"
	NewSessionTimeFieldID  = "new_session_time"
	NewSessionNotesFieldID = "new_session_notes"
	ManageNewSessionPrefix = "manage_new_session"

	SessionResponseAcceptPrefix  = "session_response_accept"
	SessionResponseDeclinePrefix = "session_response_decline"
	SessionResponseConfirmPrefix = "session_response_confirm"
	SessionResponseCancelPrefix  = "session_response_cancel"
	SessionResponseRetractPrefix = "session_response_retract"

	SessionConflictPrefix    = "session_conflict"
	SessionConflictSelPrefix = "session_conflict_sel"
)
const (
	// user-facing
	NewSessionCommandDesc      = "Schedule and announce a new session for your campaign."
	NewSessionOptionDesc       = "Campaign to schedule a session for"
	NewSessionModalTitle       = "📅 Schedule a New Session"
	NewSessionNotesLabel       = "What to expect (optional)"
	NewSessionNotesPlaceholder = "e.g. Picking up from last time- don't forget your character sheet!"
	ManageNewSessionLabel      = "📅 New Session"

	SessionEmbedGoingFmt = "✅ Going: %d  ·  ❌ Not Going: %d"

	SessionResponseAcceptLabel  = "✅ Going"
	SessionResponseDeclineLabel = "❌ Not Going"
	SessionResponseRetractLabel = "↩️ Retract"

	SessionResponseAcceptedMsg      = "✅ You're in! The DM has been notified."
	SessionResponseDeclinedMsg      = "❌ Can't make it- the DM has been notified."
	SessionResponseWaitlistedMsg    = "⏳ Session is full: you're on the waitlist. The DM will confirm the final party."
	SessionResponseConflictFmt      = "⚠️ You already have a session at this time: **%s** on %s.\nConfirm anyway?"
	SessionResponseConfirmLabel     = "Confirm anyway"
	SessionResponseCancelLabel      = "Cancel"
	SessionResponseRetractedMsg     = "↩️ Your session slot has been retracted. The DM has been notified."
	SessionResponseCooldown         = "⏳ You just responded. Try again in %s."
	SessionResponseRetractUsed      = "ℹ️ You can only retract your response once per session."
	SessionResponseRetractNone      = "ℹ️ You haven't given a response to this session yet."
	SessionResponseDMNotifyRetract  = "↩️ <@%s> retracted their session assistance response for **%s** · %s"
	SessionResponseWaitlistPromoted = "✅ A spot opened up for **%s** · %s. You have been moved out of the waiting list. You can play now!"
	SessionResponseAlreadySet       = "ℹ️ You've already responded to this session."
	SessionResponseNotMember        = "⚠️ Join this campaign before responding to its sessions."
	SessionResponseGone             = "ℹ️ This session is no longer active."

	SessionResponseDMNotifyAccept         = "✅ <@%s> is going to **%s** · %s"
	SessionResponseDMNotifyAcceptConflict = "✅ <@%s> is going to **%s** · %s ⚠️ They have a scheduling conflict: going to **%s** at the same time."
	SessionResponseDMNotifyDecline        = "❌ <@%s> can't make it for **%s** · %s"
	SessionResponseDMNotifyWaitlist       = "⏳ <@%s> is on the waitlist for **%s** · %s (session full)"

	NewSessionAnnouncedFmt    = "✅ Session for **%s** announced- %d member(s) notified."
	NewSessionNoChannel       = "⚠️ This campaign has no channel. Set one up in campaign settings first."
	NewSessionDMContentFmt    = "📅 **%s**- New Session!\n<t:%d:F> · <t:%d:R>%s"
	SessionReminderContentFmt = "⏰ Reminder: **%s** starts in ~1 hour!\n<t:%d:F>%s"

	SessionEmbedTitleFmt           = "📅 New Session- %s"
	SessionEmbedGoingLabel         = "✅ Going"
	SessionEmbedNotGoingLabel      = "❌ Not Going"
	SessionEmbedWaitlistedLabel    = "⏳ Waitlisted"
	SessionResponseCancelledMsg    = "ℹ️ Response cancelled."
	SessionResponseLineEmptyFmt    = "%s (0):-"
	SessionResponseLineFmt         = "%s (%d): %s"
	SessionResponseLineOverflowFmt = " +%d more"

	SessionConflictButtonLabel  = "⚠️ Schedule conflict"
	SessionConflictPrompt       = "You have a conflict with another session around this time. Which one would you want to go?"
	SessionConflictNone         = "ℹ️ No overlapping sessions found for you right now."
	SessionConflictDMToAbsent   = "⚠️ <@%s> has a schedule conflict and won't attend **%s**'s session (<t:%d:F>). They're playing in **%s** instead."
	SessionConflictDMToPresent  = "ℹ️ <@%s> intends to attend **%s**'s session (<t:%d:F>) over **%s**. You need to adjust the party."
	SessionConflictConfirmedFmt = "✅ Both DMs have been notified. Your response for **%s** has been set to not going."
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
	PlayerSetSheetSuccess           = "✅ Character sheet updated for **%s**."
	PlayerSetSheetFailed            = "🚫 Failed to save character sheet."
	PlayerSetSheetLabel             = "📜 Set Sheet"
	PlayerOpenSheetLabel            = "📰 Open Sheet"
	PlayerSetTokenLabel             = "🐺 Set Token"
	PlayerTokenSelectPlaceholder    = "Select a token..."
	PlayerTokenNewHint              = "Use `/newtoken` to create a new one."
	PlayerTokenAssignLabel          = "🏰 Assign to Campaign"
	PlayerTokenAssignSuccess        = "✅ Token assigned to **%s**."
	PlayerNoTokens                  = "ℹ️ No tokens found. Create one first with `/newtoken`."
	PlayerLeaveConfirmLabel         = "⬅️ Yes, Leave"
	PlayerLeaveConfirmMsg           = "⚠️ Are you sure you want to leave **%s**? This cannot be undone."
	PlayerLeaveCancelLabel          = "❌ Cancel"
	PlayerContactDMModalTitle       = "📨 Send a Message to Your DM"
	PlayerContactDMFieldLabel       = "✉️ Message"
	PlayerContactDMFieldPlaceholder = "Ask about scheduling, character questions, lore..."
	PlayerContactDMLabel            = "📨 Contact DM"
	PlayerContactDMSuccess          = "✅ Your message has been sent to the DM."
	PlayerContactDMReceived         = "**🗣️ Message from <@%s>** regarding **%s**:\n\n%s"

	PlayerDownloadTokensLabel       = "⬇️ Download Tokens"
	PlayerDownloadSelectPlaceholder = "Select a campaign..."
	PlayerDownloadAllLabel          = "🏰 All campaigns"
	PlayerDownloadNoTokens          = "ℹ️ You haven't assigned tokens to any campaigns yet. Open a campaign card and use **Set Token** to assign one."
	PlayerDownloadContent           = "ℹ️ Here are your %d token(s):"
)

// Campaign import
const (
	ImportCampaignCommandName = "importcampaign"
	ImportCampaignCommandDesc = "Import an existing campaign channel and role into Moontracer"

	ImportCampaignOptChannel = "channel"
	ImportCampaignOptRole    = "role"
	ImportCampaignOptDM      = "dm"

	// Custom-ID prefixes for the one-step thread-mapping flow.
	ImportThreadSelPrefix = "import_thread_sel" // import_thread_sel:<sessionID>:<threadName>
	ImportConfirmPrefix   = "import_confirm"    // import_confirm:<sessionID>
	ImportCancelPrefix    = "import_cancel"     // import_cancel:<sessionID>

	// Sentinel stored in a session mapping when the user wants the bot to create the thread.
	ImportCreateNew = "new"

	// Step header.
	ImportStep1Header = "**Map threads for #%s** (core threads)\nChoose an existing thread for each slot, or leave as **Create new**."

	// Select-menu placeholders (core threads only).
	ImportSelWelcome       = "🐺 Welcome!"
	ImportSelAnnouncements = "🗣️ Announcements"
	ImportSelSessions      = "📅 Sessions"
	ImportSelDiceRolls     = "🎲 Dice rolls"

	// "Create new" option shown at the top of every select menu.
	ImportOptCreateNew      = "Create new"
	ImportOptCreateNewDescr = "Bot will create this thread automatically."

	// Button labels.
	ImportConfirmLabel = "✅ Confirm Import"
	ImportCancelLabel  = "❌ Cancel"

	// Terminal messages.
	ImportCampaignProcessing = "⏳ Fetching threads, just a moment…"
	ImportCampaignSuccess    = "✅ Imported **%s**: %d member(s) registered, %d thread(s) bound, %d thread(s) created."
	ImportCampaignCancelled  = "⚠️ Import cancelled."
	ImportCampaignErrDB      = "❌ Failed to write campaign to the database."
	ImportCampaignErrChannel = "❌ Could not read the channel. Make sure the bot has access to it."
	ImportCampaignErrSession = "❌ This import session has expired. Please run /importcampaign again."
)

// Forum post (all user-facing)
const (
	ForumPostFormatCampaign  = "Campaign"
	ForumPostFormatOneshot   = "One-shot"
	ForumPostFormatWestmarch = "Westmarch"
	ForumPostScheduleUnset   = "Unset"
	ForumPostStatusOpen      = "Open"
	ForumPostStatusClosed    = "Closed"
	ForumPostSlotsUnlimited  = "Unlimited"
	ForumPostNoPlayers       = "*None yet*"
)

// Billboard

// Campaign billboard: forum channel names (internal)
const (
	BillboardChannelCampaign  = "new-campaigns"
	BillboardChannelOneshot   = "one-shots"
	BillboardChannelWestmarch = "westmarches"
)

/*
BillboardPinMessage is pinned in the campaign channel after the forum thread is created.

%s is the billboard thread/channel ID (rendered as a clickable channel mention).
*/
const BillboardPinMessage = "**About this campaign:** <#%s>"

/*
CampaignAnnouncementThreadFmt is appended to the campaign channel announcement when
a billboard thread exists.

%s is the thread ID.
*/
const CampaignAnnouncementThreadFmt = "🏰 **About this campaign:** <#%s>"

/*
AnnouncementDMFmt is the DM sent to each campaign member when the DM posts in the announcements thread.

Args: campaign name, DM user ID, message content.
*/
const AnnouncementDMFmt = "🗣️ **[%s]** <@%s> says:\n\n%s"

// New campaign modal: schedule step (internal)
const (
	NewCampaignWarningsFieldID = "newcampaign_warnings"
)

// New campaign modal: schedule step (user-facing)
const (
	NewCampaignWarningsLabel       = "Content warnings (optional)"
	NewCampaignWarningsPlaceholder = "e.g. Violence, horror (comma-separated)"
)

// New campaign modal: game details step (internal)
const (
	NewCampaignGameDetailsModalID = "modal_newcampaign_gamedetails"
	NewCampaignRulesFieldID       = "newcampaign_rules"
	NewCampaignVTTFieldID         = "newcampaign_vtt"
	NewCampaignBooksFieldID       = "newcampaign_books"
	NewCampaignExtraFieldID       = "newcampaign_extra"
)

// New campaign modal: game details step (user-facing)
const (
	NewCampaignGameDetailsModalTitle = "🎲 Game Details"
	NewCampaignRulesLabel            = "House rules / variant (optional)"
	NewCampaignRulesPlaceholder      = "e.g. Milestone levelling, no flanking..."
	NewCampaignVTTLabel              = "VTT platform (optional)"
	NewCampaignVTTPlaceholder        = "e.g. Roll20, Foundry VTT, Owlbear Rodeo..."
	NewCampaignBooksLabel            = "Books allowed (comma-separated, optional)"
	NewCampaignBooksPlaceholder      = "e.g. PHB, Xanathar's, Tasha's"
	NewCampaignExtraLabel            = "Extra info for players (optional)"
	NewCampaignExtraPlaceholder      = "Tone, session zero notes, Discord links..."
)

// Import campaign: billboard channel selector (user-facing)
const (
	ImportBillboardSelPrefix      = "import_billboard_sel"
	ImportBillboardSelPlaceholder = "Select the billboard forum channel…"
	ImportBillboardPrompt         = "**Select the forum channel** where this campaign's post should appear, or skip to auto-create one."
	ImportBillboardSkipPrefix     = "import_billboard_skip"
	ImportBillboardSkipLabel      = "Auto-create"
)

// Timezone preference
const (
	// identifiers
	TimezonePrefix   = "set_timezone"
	TimezoneSelectID = "timezone_select"
)
const (
	// user-facing
	TimezoneLabel             = "🌐 Timezone"
	TimezoneHeader            = "**Set your timezone**\nTimes will be shown in your local time.\nCurrent: **%s**"
	TimezoneSelectPlaceholder = "Select your timezone…"
	TimezoneSuccess           = "✅ Timezone set to **%s**."
	TimezoneInvalid           = "⚠️ Unknown timezone. Please select from the list."
)

// Campaign role naming

// standard discord colors
var DiscordRoleColors = []int{
	0x1ABC9C, 0x2ECC71, 0x3498DB, 0x9B59B6, 0xE91E63,
	0xF1C40F, 0xE67E22, 0xE74C3C, 0x11806A, 0x1F8B4C,
	0x206694, 0x71368A, 0xAD1457, 0xC27C0E, 0xA84300,
	0x992D22, 0x607D8B, 0x99AAB5,
}

// RandomRoleColor returns a random color from the Discord role palette.
func RandomRoleColor() int {
	return DiscordRoleColors[rand.Intn(len(DiscordRoleColors))]
}

/*
CampaignRoleName derives a short role name from a campaign name.

Truncates at the first separator (":", " - ") or connector word in
English ("of", "and", "or", ...) or Spanish ("de", "del", "y", "o", ...).
*/
func CampaignRoleName(name string) string {
	lower := strings.ToLower(name)
	cutAt := len(name)

	for _, sep := range []string{":", " - "} {
		if idx := strings.Index(lower, sep); idx != -1 && idx < cutAt {
			cutAt = idx
		}
	}
	connectors := []string{
		// EN
		" and ", " or ", " of ", " the ", " a ", " an ",
		" in ", " into ", " on ", " at ", " to ", " for ",
		" with ", " by ", " from ", " vs ", " vs. ",

		// ES
		" de ", " del ", " y ", " o ", " en ", " con ",
		" por ", " para ", " desde ", " hasta ", " entre ",
		" sobre ", " hacia ", " sin ",
	}
	for _, conn := range connectors {
		if idx := strings.Index(lower, conn); idx != -1 && idx < cutAt {
			cutAt = idx
		}
	}

	result := strings.TrimSpace(name[:cutAt])
	if len(result) < 3 {
		return strings.TrimSpace(name)
	}
	return result
}
