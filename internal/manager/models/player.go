package models

import "github.com/uptrace/bun"

// Player represents a Discord user participating in campaigns.
// Masters (DMs) are players with a "dm" role in CampaignPlayer. There is no
// separate Master entity, as Masters are players too.
type Player struct {
	bun.BaseModel `bun:"table:players"`

	// Discord user ID, used as primary key.
	ID string `bun:",pk,notnull" json:"id"`

	// Has-many relations.
	Tokens          []Token          `bun:"rel:has-many,join:id=owner_id" json:"tokens,omitempty"`
	CampaignPlayers []CampaignPlayer `bun:"rel:has-many,join:id=player_id" json:"campaign_players,omitempty"`
}
