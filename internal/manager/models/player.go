package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// ServerRole represents a player's guild-wide role (not campaign-specific).
// Roles are hierarchical: Admin > Mod > Player.
// Admin implies Mod. Mod can also be a DM of campaigns (roles are additive).
type ServerRole string

const (
	ServerRolePlayer ServerRole = "player"
	ServerRoleMod    ServerRole = "mod"
	ServerRoleAdmin  ServerRole = "admin"
)

// Weight returns the numeric rank of a ServerRole for hierarchy comparisons.
// Higher weight = higher authority. Used by ban protection to ensure
// a user can only ban someone with a strictly lower role.
func (r ServerRole) Weight() int {
	switch r {
	case ServerRoleAdmin:
		return 2
	case ServerRoleMod:
		return 1
	default:
		return 0
	}
}

// AuditAction describes what moderation operation was performed.
type AuditAction string

const (
	AuditBan             AuditAction = "ban"
	AuditUnban           AuditAction = "unban"
	AuditCampaignBan     AuditAction = "campaign_ban"
	AuditCampaignArchive   AuditAction = "campaign_archive"
	AuditSessionReschedule AuditAction = "session_reschedule"
)

// AuditEntry is an immutable moderation log record.
type AuditEntry struct {
	bun.BaseModel `bun:"table:audit_entries"`

	ID        int64       `bun:",pk,autoincrement" json:"id"`
	PlayerID  string      `bun:",notnull" json:"player_id"`
	Player    *Player     `bun:"rel:belongs-to,join:player_id=id" json:"player,omitempty"`
	AuthorID  string      `bun:",notnull" json:"author_id"`
	Author    *Player     `bun:"rel:belongs-to,join:author_id=id" json:"author,omitempty"`
	Action    AuditAction `bun:",notnull" json:"action"`
	Reason    string      `bun:",nullzero" json:"reason,omitempty"`
	CreatedAt time.Time   `bun:",notnull,default:current_timestamp" json:"created_at"`
}

// InsertAuditEntry writes an immutable moderation log record.
func InsertAuditEntry(db *bun.DB, playerID, authorID string, action AuditAction, reason string) error {
	ctx := context.Background()
	entry := &AuditEntry{
		PlayerID:  playerID,
		AuthorID:  authorID,
		Action:    action,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	_, err := db.NewInsert().Model(entry).Exec(ctx)
	return err
}

// GetPlayerWithCampaigns loads a Player by ID with its CampaignPlayers relation.
// This is required for Player methods that inspect campaign memberships
// (IsDMOf, IsMemberOf, IsBannedFromCampaign, etc.).
func GetPlayerWithCampaigns(db *bun.DB, playerID string) (*Player, error) {
	ctx := context.Background()
	var player Player
	err := db.NewSelect().
		Model(&player).
		Relation("CampaignPlayers").
		Where("player.id = ?", playerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

// Player represents a Discord user participating in campaigns.
// Player is the single owner of all role and permission data.
// DM status is campaign-scoped via CampaignPlayer, while Mod/Admin are server-wide via Role.
type Player struct {
	bun.BaseModel `bun:"table:players"`

	ID              string           `bun:",pk,notnull" json:"id"`                                   // Discord user ID, used as primary key.
	Role            ServerRole       `bun:",notnull,default:'player'" json:"role"`                   // Guild-wide role, synced from Discord. Determines mod/admin permissions.
	PlayerIsBanned  bool             `bun:"column:is_banned,notnull,default:false" json:"is_banned"` // Global ban. A globally banned player cannot interact with the bot at all.
	PlayerBanReason string           `bun:"column:ban_reason,nullzero" json:"ban_reason,omitempty"`
	Media           []Media          `bun:"rel:has-many,join:id=owner_id" json:"media,omitempty"`
	CampaignPlayers []CampaignPlayer `bun:"rel:has-many,join:id=player_id" json:"campaign_players,omitempty"`
	ModerationLog   []AuditEntry     `bun:"rel:has-many,join:id=player_id" json:"moderation_log,omitempty"`
}

// IsAdmin returns true if the player has the admin server role.
func (p *Player) IsAdmin() bool {
	return p.Role == ServerRoleAdmin
}

// IsMod returns true if the player is a mod or admin (admin is a superset of mod).
func (p *Player) IsMod() bool {
	return p.Role == ServerRoleMod || p.Role == ServerRoleAdmin
}

// IsDMOf returns true if the player is the DM of the given campaign.
// Requires CampaignPlayers to be loaded (via Relation).
func (p *Player) IsDMOf(campaignID string) bool {
	for _, cp := range p.CampaignPlayers {
		if cp.CampaignID == campaignID && cp.Role == RoleDM {
			return true
		}
	}
	return false
}

// IsMemberOf returns true if the player is an active member of the given campaign.
// Requires CampaignPlayers to be loaded (via Relation).
func (p *Player) IsMemberOf(campaignID string) bool {
	for _, cp := range p.CampaignPlayers {
		if cp.CampaignID == campaignID && cp.Status == StatusActive {
			return true
		}
	}
	return false
}

// DMCampaignIDs returns the IDs of all campaigns this player DMs.
// Requires CampaignPlayers to be loaded (via Relation).
func (p *Player) DMCampaignIDs() []string {
	var ids []string
	for _, cp := range p.CampaignPlayers {
		if cp.Role == RoleDM {
			ids = append(ids, cp.CampaignID)
		}
	}
	return ids
}

// IsBannedFrom returns the CampaignPlayer relationships on which the Player is banned from.
// Requires CampaignPlayers to be loaded (via Relation).
func (p *Player) IsBannedFrom() []CampaignPlayer {
	var bannedFromCampaigns []CampaignPlayer

	for _, cp := range p.CampaignPlayers {
		if cp.PlayerID == p.ID && cp.BannedFromCampaign {
			bannedFromCampaigns = append(bannedFromCampaigns, cp)
		}
	}

	return bannedFromCampaigns
}

// IsBannedFromCampaign returns true if the player has a campaign-scoped ban on the given campaign.
// Requires CampaignPlayers to be loaded (via Relation).
func (p *Player) IsBannedFromCampaign(campaignID string) bool {
	for _, cp := range p.CampaignPlayers {
		if cp.CampaignID == campaignID && cp.BannedFromCampaign {
			return true
		}
	}
	return false
}
