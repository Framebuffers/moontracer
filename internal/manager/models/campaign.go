package models

import "time"

// Campaign represents a DnD campaign.
type Campaign struct {
	// Server-generated unique identifier. This is the ID that will be used throughout to identify this particular campaign.
	ID string `json:"id"`

	// Player FK of the user (DM) who created the campaign.
	DungeonMaster string `json:"dm"`

	// Campaign's description, synopsis, details, lore, or whatever you want to add to describe your setting.
	Description string `json:"description"`

	// Embedded singular game configuration. Here is where most of your campaign's technical/gameplay details are.
	Game GameConfig `json:"game"`

	// Details about your campaign, like open slots, the style, trigger warnings, extra info by the DM to be added to the Campaign's description.
	Slots     int      `json:"slots"`
	IsOpen    bool     `json:"is_open"`
	IsOneshot bool     `json:"is_oneshot"`
	Warnings  []string `json:"warnings,omitempty"`
	Extra     string   `json:"extra,omitempty"`

	// Campaign schedule. This will define your player's availability.
	Schedule CampaignSchedule `json:"schedule"`

	// Links and media
	Links         []string `json:"links,omitempty"`
	VTTLink       string   `json:"vtt_link,omitempty"`
	CampaignMedia []string `json:"campaign_media,omitempty"` // base64-encoded images
}

// GameConfig describes the rule system and tools used by a campaign.
// Embedded as a singular struct inside Campaign (a campaign has exactly one
// game configuration).
type GameConfig struct {
	Edition      string   `json:"edition"` // e.g. "5e", "pathfinder", "other"
	Rules        string   `json:"rules"`   // e.g. "2014", "2024", "homebrew"
	VTT          string   `json:"vtt"`     // e.g. "owlbear", "foundry", "custom"
	BooksAllowed []string `json:"books_allowed,omitempty"`
	Other        []string `json:"other,omitempty"`
}

// CampaignSchedule describes when, how often and when was the last time a Campaign has been played.
// For Players, this is used to create their own schedules, avoid clashes between sessions, etc; avoiding DMs having players bail out of sessions because of time clashes.
type CampaignSchedule struct {
	Frequency   CampaignFrequency `json:"frequency"`
	CreatedAt   time.Time         `json:"created_at"`
	LastSession time.Time         `json:"last_session"`
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
