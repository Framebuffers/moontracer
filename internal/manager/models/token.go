package models

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
	// Server-generated unique identifier.
	ID string `json:"id"`

	// FK to the Player who owns this token.
	OwnerID string `json:"owner_id"`

	// Image or text content (base64 or plain string).
	Content string `json:"content"`

	Type TokenType `json:"type"`
	Name string    `json:"name"`
}
