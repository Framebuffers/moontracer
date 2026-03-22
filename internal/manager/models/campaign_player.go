package models

import (
	"context"

	"github.com/uptrace/bun"
)

// Role distinguishes players from DMs within a campaign.
// Players can be DMs.
type Role string

// Permissions are hierarchical: admins can override mod permissions, mod can override DMs and so on.
// e.g. to authorise a campaign, a DM has to ask a mod for permission.
// But, if a mod account gets compromised, an admin can override any mod and act as the final decisionmaker of the whole permissions chain.
// Every new member starts as a player, with the server admin as the only admin account.
// On Moontracer's config, a Moderator role can be used to automatically give mod access to the bot.
// A player becomes a DM when: 1) uploads a request for a new campaign (filling the new campaign form), and 2) gets authorised by a mod.
// A player can be part of a Campaign if:
//  1. the campaign is active and properly authorised by a mod/admin.
//  2. the DM is active, authorised as such, owner of the campaign and is not banned.
const (
	RolePlayer    Role = "player"
	RoleDM        Role = "dm"
	RoleModerator Role = "mod"
	RoleAdmin     Role = "admin"
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

	PlayerID              string               `bun:",pk,notnull" json:"player_id"`
	Player                *Player              `bun:"rel:belongs-to,join:player_id=id" json:"player,omitempty"`
	CampaignID            string               `bun:",pk,notnull" json:"campaign_id"`
	Campaign              *Campaign            `bun:"rel:belongs-to,join:campaign_id=id" json:"campaign,omitempty"`
	Role                  Role                 `bun:",notnull,default:'player'" json:"role"`
	TokenID               string               `bun:",nullzero" json:"token_id,omitempty"`
	Token                 *Token               `bun:"rel:belongs-to,join:token_id=id" json:"token,omitempty"`
	Status                CampaignPlayerStatus `bun:",notnull,default:'active'" json:"status"`
	SessionsPlayed        int                  `bun:",notnull,default:0" json:"sessions_played"`
	DiceThrowPicture      string               `bun:",nullzero" json:"dice_throw_picture,omitempty"`
	BanReason             string               `bun:",nullzero" json:"ban_reason,omitempty"`
	BannedFromCampaign    bool                 `bun:",notnull,default:false" json:"banned_from_campaign"`
	BanReasonFromCampaign string               `bun:",nullzero" json:"ban_reason_from_campaign,omitempty"`
}

// GetCampaignPlayers retrieves all CampaignPlayers belonging to a given Campaign ID
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

// GetPlayerCampaigns retrieves all the Campaigns a Player belongs to.
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

// RemoveCampaignPlayer removes a player from a given Campaign.
func RemoveCampaignPlayer(db *bun.DB, playerID, campaignID string) error {
	ctx := context.Background()
	_, err := db.NewDelete().Model((*CampaignPlayer)(nil)).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Exec(ctx)
	return err
}

// SetCampaignPlayerStatus sets the status of a Player to any CampainPlayerStatus entry.
func SetCampaignPlayerStatus(db *bun.DB, playerID, campaignID string, status CampaignPlayerStatus) error {
	ctx := context.Background()
	_, err := db.NewUpdate().Model((*CampaignPlayer)(nil)).
		Set("status = ?", status).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Exec(ctx)
	return err
}

// BulkSetCampaignPlayerStatus updates campaign memberships for a player in bulk.
// skipLogic decides which entries to leave untouched (nil means skip nothing).
// Returns updated/skipped counts and a map of campaignID→error for any failures.
func BulkSetCampaignPlayerStatus(db *bun.DB, playerID string, campaigns []CampaignPlayer, to CampaignPlayerStatus, skipLogic func(CampaignPlayer) bool) (updated int, skipped int, errs map[string]error) {
	errors := make(map[string]error)

	for _, cp := range campaigns {
		if skipLogic != nil && skipLogic(cp) {
			skipped++
			continue
		}
		if err := SetCampaignPlayerStatus(db, playerID, cp.CampaignID, to); err != nil {
			errors[cp.CampaignID] = err
			continue
		}
		updated++
	}

	if len(errors) == 0 {
		return updated, skipped, nil
	}
	return updated, skipped, errors
}
