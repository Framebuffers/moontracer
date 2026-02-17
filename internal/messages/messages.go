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
	IDCommandName          = "id"
	IDCommandDesc          = "Campaign ID to look up."
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
	OpenCampaignLabel          = "Open Campaign"
	ClosedCampaignLabel        = "Closed Campaign"
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
	FieldDescriptionID = "description"
	FieldEditionID     = "edition"
	FieldRulesID       = "rules"
	FieldSlotsID       = "slots"
	FieldWarningsID    = "warnings"
)

// New campaign modal labels
const (
	FieldDescriptionLabel = "Description"
	FieldEditionLabel     = "Edition"
	FieldRulesLabel       = "Rules"
	FieldSlotsLabel       = "Player Slots"
	FieldWarningsLabel    = "Content Warnings (comma-separated)"
)

// New campaign modal placeholders
const (
	FieldDescriptionPlaceholder = "Describe your campaign setting and premise..."
	FieldEditionPlaceholder     = "e.g. 5e, 3.5e, PF2e"
	FieldRulesPlaceholder       = "e.g. 2024, 2014, homebrew"
	FieldSlotsPlaceholder       = "e.g. 4"
	FieldWarningsPlaceholder    = "e.g. Violence, Horror, Permadeath"
)
