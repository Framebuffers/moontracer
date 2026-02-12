package models

import "github.com/uptrace/bun"

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
	bun.BaseModel `bun:"table:campaign_players"`

	PlayerID   string               `bun:",pk,notnull" json:"player_id"`
	Player     *Player              `bun:"rel:belongs-to,join:player_id=id" json:"player,omitempty"`

	CampaignID string               `bun:",pk,notnull" json:"campaign_id"`
	Campaign   *Campaign            `bun:"rel:belongs-to,join:campaign_id=id" json:"campaign,omitempty"`

	Role       CampaignPlayerRole   `bun:",notnull,default:'player'" json:"role"`

	TokenID    string               `bun:",nullzero" json:"token_id,omitempty"`
	Token      *Token               `bun:"rel:belongs-to,join:token_id=id" json:"token,omitempty"`

	Status     CampaignPlayerStatus `bun:",notnull,default:'active'" json:"status"`

	SessionsPlayed   int    `bun:",notnull,default:0" json:"sessions_played"`
	DiceThrowPicture string `bun:",nullzero" json:"dice_throw_picture,omitempty"`
}
