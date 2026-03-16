package models

import "github.com/uptrace/bun"

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

// Player represents a Discord user participating in campaigns.
// Player is the single owner of all role and permission data.
// DM status is campaign-scoped via CampaignPlayer, while Mod/Admin are server-wide via Role.
type Player struct {
	bun.BaseModel `bun:"table:players"`

	// Discord user ID, used as primary key.
	ID string `bun:",pk,notnull" json:"id"`

	// Guild-wide role, synced from Discord. Determines mod/admin permissions.
	Role ServerRole `bun:",notnull,default:'player'" json:"role"`

	// Global ban. A globally banned player cannot interact with the bot at all.
	IsBanned  bool   `bun:",notnull,default:false" json:"is_banned"`
	BanReason string `bun:",nullzero" json:"ban_reason,omitempty"`

	// Has-many relations.
	Tokens          []Token          `bun:"rel:has-many,join:id=owner_id" json:"tokens,omitempty"`
	CampaignPlayers []CampaignPlayer `bun:"rel:has-many,join:id=player_id" json:"campaign_players,omitempty"`
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
