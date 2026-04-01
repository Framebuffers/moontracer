package models

import "github.com/uptrace/bun"

// TokenType classifies a token's purpose.
// Tokens are used in the VTT as a representation of characters, NPCs or assets.
type TokenType string

const (
	TokenPlayer TokenType = "player"
	TokenNPC    TokenType = "npc"
	TokenAsset  TokenType = "asset"
)

// Token represents a visual token (character art, NPC image, map asset, etc.)
// owned by a player.
// Assignment to a specific campaign is tracked in CampaignPlayer.TokenID
type Token struct {
	bun.BaseModel `bun:"table:tokens"`

	// Server-generated unique identifier.
	ID string `bun:",pk,notnull" json:"id"`

	// FK to the Player who owns this token.
	OwnerID string  `bun:",notnull" json:"owner_id"`
	Owner   *Player `bun:"rel:belongs-to,join:owner_id=id" json:"owner,omitempty"`

	// Image or text content (base64 or plain string).
	Content string `bun:",notnull" json:"content"`

	Type TokenType `bun:",notnull" json:"type"`
	Name string    `bun:",notnull" json:"name"`
}
