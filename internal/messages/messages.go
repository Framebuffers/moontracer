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
	UnbanCommandName        = "unban"
	UnbanCommandDesc        = "Lift a global ban from a player."
	UnbanTargetNotFound     = "That player is not registered."
	UnbanTargetNotBanned    = "That player is not banned."
	UnbanFailureMessage     = "Failed to unban player."
	UnbanSuccessMessage     = "Unbanned <@%s>. They can interact with the bot again."
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
	ManageRescheduleLabel = "Reschedule"
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
	CampaignApprovePrefix      = "campaign_approve"
	CampaignDenyPrefix         = "campaign_deny"
	CampaignDenyModalPrefix    = "campaign_deny_modal"
	ApproveButtonLabel         = "Approve"
	DenyButtonLabel            = "Deny"
	CampaignApprovedMessage    = "Campaign **%s** has been approved."
	CampaignDeniedMessage      = "Campaign **%s** has been denied and deleted."
	CampaignApproveNotModError = "Only mods or admins can approve campaigns."
	CampaignApproveNotFound    = "Campaign not found or already processed."
	CampaignApproveError       = "Failed to process campaign approval."
	CampaignDenyModalTitle     = "Deny Campaign"
	CampaignDenyReasonLabel    = "Reason"
	CampaignDenyReasonPlaceholder = "Why is this campaign being denied?"
	CampaignDenyReasonFieldID  = "deny_reason"
	CampaignDeniedDMMessage    = "Your campaign **%s** has been denied. Reason: %s"
	CampaignApprovedDMMessage  = "Your campaign **%s** has been approved!"
)

// Help command
const (
	HelpCommandName = "help"
	HelpCommandDesc = "Get a list of all available commands."
)
