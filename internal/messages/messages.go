package messages

// Generic
const (
	GenericErrorMessage = "Something went wrong."
	InvalidButtonDataMessage = "Invalid button data."
)

// Registration
const (
	NotRegisteredMessage        = "You need to `/register` first."
	AlreadyRegisteredMessage    = "You are already registered!"
	RegistrationFailureMessage  = "Failed to register. Please try again later."
	RegistrationSuccessMessage  = "Welcome, <@%s>! You are now registered."
	RegistrationCheckError      = "register: error checking registration: "
	RegistrationInsertError     = "register: error inserting player: "
)

// Campaign lookup
const (
	CampaignNotFoundMessage        = "Campaign not found."
	CampaignLoadFailureErrorMessage = "Failed to load campaign."
	CampaignPlayersLoadError       = "Failed to load campaign players."
	CampaignFetchError             = "campaign: error fetching campaign %s: "
	PlayerFetchErrorMessage        = "models.GetCampaignPlayers(): Error fetching players: "
)

// Campaign creation
const (
	SlotCountMismatchErrorMessage             = "Invalid slot count. Please enter a number greater than 0."
	CampaignCreationFailureErrorMessage       = "campaign.CreateCampaign(): error creating campaign: "
	CampaignAndRegistrationFailureErrorMessage = "Failed to create campaign. Make sure you are registered."
	CampaignCreationMessage                   = "You just created a new campaign: "
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
	NoCampaignsMessage       = "You are not in any campaigns yet."
	MyCampaignsLoadError     = "Failed to load your campaigns."
)

// New campaign modal
const (
	NewCampaignModalError = "newcampaign: error opening modal: "
)
