package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Campaign represents a DnD campaign.
type Campaign struct {
	bun.BaseModel `bun:"table:campaigns"`

	// Server-generated unique identifier. This is the ID that will be used throughout to identify this particular campaign.
	ID string `bun:",pk,notnull" json:"id"`

	// Player FK of the user (DM) who created the campaign.
	DungeonMaster string  `bun:",notnull" json:"dm"`
	DM            *Player `bun:"rel:belongs-to,join:dungeon_master=id" json:"dm_player,omitempty"`

	// Campaign's description, synopsis, details, lore, or whatever you want to add to describe your setting.
	Description string `bun:",notnull,default:''" json:"description"`

	// Embedded singular game configuration. Stored as flat columns.
	Game GameConfig `bun:"embed:"`

	// Details about your campaign, like open slots, the style, trigger warnings, extra info by the DM to be added to the Campaign's description.
	Slots     int      `bun:",notnull,default:0" json:"slots"`
	IsOpen    bool     `bun:",notnull,default:false" json:"is_open"`
	IsOneshot bool     `bun:",notnull,default:false" json:"is_oneshot"`
	Warnings  []string `bun:",array,type:jsonb" json:"warnings,omitempty"`
	Extra     string   `bun:",default:''" json:"extra,omitempty"`

	// Campaign schedule.
	Schedule CampaignSchedule `bun:"embed:"`

	// Links and media
	Links         []string `bun:",array,type:jsonb" json:"links,omitempty"`
	VTTLink       string   `bun:",default:''" json:"vtt_link,omitempty"`
	CampaignMedia []string `bun:",array,type:jsonb" json:"campaign_media,omitempty"`

	// Has-many relation.
	CampaignPlayers []CampaignPlayer `bun:"rel:has-many,join:id=campaign_id" json:"campaign_players,omitempty"`
}

// GameConfig holds the game system details for a campaign.
type GameConfig struct {
	Edition      string   `bun:",notnull,default:''" json:"edition"`
	Rules        string   `bun:",notnull,default:''" json:"rules"`
	VTT          string   `bun:",notnull,default:''" json:"vtt"`
	BooksAllowed []string `bun:",array,type:jsonb" json:"books_allowed,omitempty"`
	OtherGame    []string `bun:",array,type:jsonb" json:"other_game,omitempty"`
}

// CampaignSchedule holds the timing details for a campaign.
type CampaignSchedule struct {
	Frequency   CampaignFrequency `bun:",notnull,default:'weekly'" json:"frequency"`
	CreatedAt   time.Time         `bun:",notnull,default:current_timestamp" json:"created_at"`
	LastSession time.Time         `bun:",nullzero" json:"last_session"`
}

// CampaignFrequency defines how often will sessions in this Campaign will occur.
type CampaignFrequency string

const (
	OneShot   CampaignFrequency = "oneshot"
	Weekly    CampaignFrequency = "weekly"
	Biweekly  CampaignFrequency = "biweekly"
	Monthly   CampaignFrequency = "monthly"
	Quarterly CampaignFrequency = "quarterly"
	Yearly    CampaignFrequency = "yearly"
)
