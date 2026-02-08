package models

// CampaignPlayerRole distinguishes players from DMs within a campaign.
// Players can be DMs.
type CampaignPlayerRole string

const (
	RolePlayer CampaignPlayerRole = "player"
	RoleDM     CampaignPlayerRole = "dm"
)

// CampaignPlayerStatus tracks a participant's standing in a campaign.
// Campaigns can have four (4) stages: active, on hiatus, cancelled or finished.
// But, since player can be players as well, a fifth state, `banned`, is added.
// This way, the player <-> DM relationship is satisfied.
type CampaignPlayerStatus string

const (
	StatusActive    CampaignPlayerStatus = "active"
	StatusHiatus    CampaignPlayerStatus = "hiatus"
	StatusCancelled CampaignPlayerStatus = "cancelled"
	StatusFinished  CampaignPlayerStatus = "finished"
	StatusBanned    CampaignPlayerStatus = "banned"
)

// CampaignPlayer is the join table between Player and Campaign.
// It resolves the three-way Player-Campaign-Token relationship: a token is
// assigned to a specific player within a specific campaign.
type CampaignPlayer struct {
	PlayerID   string               `json:"player_id"`
	CampaignID string               `json:"campaign_id"`
	Role       CampaignPlayerRole   `json:"role"`
	TokenID    string               `json:"token_id,omitempty"` // nullable, TokenId is the token this player uses in this campaign
	Status     CampaignPlayerStatus `json:"status"`

	SessionsPlayed   int    `json:"sessions_played"`
	DiceThrowPicture string `json:"dice_throw_picture,omitempty"` // base64-encoded image file.
}
