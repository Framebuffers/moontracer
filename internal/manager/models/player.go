package models

// Player represents a Discord user participating in campaigns.
// Masters (DMs) are players with a "dm" role in CampaignPlayer. There is no
// separate Master entity, as Masters are players too.
type Player struct {
	// Discord user ID, used as primary key.
	ID string `json:"id"`
}
