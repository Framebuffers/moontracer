package models

import (
	"context"

	"github.com/uptrace/bun"
)

// CampaignPlayerRole distinguishes players from DMs within a campaign.
// Players can be DMs.
type CampaignPlayerRole string

const (
	RolePlayer CampaignPlayerRole = "player"
	RoleDM     CampaignPlayerRole = "dm"
)

// CampaignPlayerStatus tracks a participant's (DM or Player) standing in a campaign.
// Before a Campaign is approved to be listed as active, it remanins by default as "pending".
// After it gets approved, it becomes "active".
// A campaign then can change to be on "hiatus" (paused by the DM until further notice), "cancelled" or "finished".
// Since this type also defines the relationship between player and DM, a sixth status is added: "banned".
// This last status is used when a user is permanently banned from a campaign, satisfying the player <-> DM relationship.
type CampaignPlayerStatus string

const (
	StatusActive    CampaignPlayerStatus = "active"
	StatusHiatus    CampaignPlayerStatus = "hiatus"
	StatusCancelled CampaignPlayerStatus = "cancelled"
	StatusFinished  CampaignPlayerStatus = "finished"
	StatusBanned    CampaignPlayerStatus = "banned"
	StatusPending   CampaignPlayerStatus = "pending"
)

// CampaignPlayer is the join table between Player and Campaign.
// It resolves the three-way Player-Campaign-Token relationship: a token is
// assigned to a specific player within a specific campaign.
type CampaignPlayer struct {
	bun.BaseModel `bun:"table:campaign_players"`

	PlayerID string  `bun:",pk,notnull" json:"player_id"`
	Player   *Player `bun:"rel:belongs-to,join:player_id=id" json:"player,omitempty"`

	CampaignID string    `bun:",pk,notnull" json:"campaign_id"`
	Campaign   *Campaign `bun:"rel:belongs-to,join:campaign_id=id" json:"campaign,omitempty"`

	Role CampaignPlayerRole `bun:",notnull,default:'player'" json:"role"`

	TokenID string `bun:",nullzero" json:"token_id,omitempty"`
	Token   *Token `bun:"rel:belongs-to,join:token_id=id" json:"token,omitempty"`

	Status CampaignPlayerStatus `bun:",notnull,default:'active'" json:"status"`

	SessionsPlayed   int    `bun:",notnull,default:0" json:"sessions_played"`
	DiceThrowPicture string `bun:",nullzero" json:"dice_throw_picture,omitempty"`
}

func GetCampaignPlayers(db *bun.DB, campaignID string) ([]CampaignPlayer, error) {
	ctx := context.Background()
	var players []CampaignPlayer
	err := db.NewSelect().Model(&players).
		Relation("Player").
		Where("campaign_id = ?", campaignID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return players, nil
}

func GetPlayerCampaigns(db *bun.DB, playerID string) ([]CampaignPlayer, error) {
	ctx := context.Background()
	var campaigns []CampaignPlayer
	err := db.NewSelect().Model(&campaigns).
		Relation("Campaign").
		Where("player_id = ?", playerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return campaigns, nil
}

func RemoveCampaignPlayer(db *bun.DB, playerID, campaignID string) error {
	ctx := context.Background()
	_, err := db.NewDelete().Model((*CampaignPlayer)(nil)).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Exec(ctx)
	return err
}

func SetCampaignPlayerStatus(db *bun.DB, playerID, campaignID string, status CampaignPlayerStatus) error {
	ctx := context.Background()
	_, err := db.NewUpdate().Model((*CampaignPlayer)(nil)).
		Set("status = ?", status).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Exec(ctx)
	return err
}
