package models

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Campaign represents a DnD campaign.
type Campaign struct {
	bun.BaseModel `bun:"table:campaigns"`

	ID            string     `bun:",pk,notnull" json:"id"`      // Server-generated unique identifier. This is the ID that will be used throughout to identify this particular campaign.
	Name          string     `bun:",notnull" json:"name"`       // Display name for the campaign.
	Tag           string     `bun:",unique,notnull" json:"tag"` // Short, user-facing identifier for lookups (e.g. "strahd", "avalon", "itzaal", "nuevosur", "suvachi").
	DungeonMaster string     `bun:",notnull" json:"dm"`         // Player FK of the user (DM) who created the campaign.
	DM            *Player    `bun:"rel:belongs-to,join:dungeon_master=id" json:"dm_player,omitempty"`
	Description   string     `bun:",notnull,default:''" json:"description"` // Campaign's description, synopsis, details, lore, or whatever you want to add to describe your setting.
	Game          GameConfig `bun:"embed:"`                                 // Embedded singular game configuration. Stored as flat columns.

	// Details about your campaign, like open slots, the style, trigger warnings, extra info by the DM to be added to the Campaign's description.

	Slots           int      `bun:",notnull,default:0" json:"slots"`            // Total roster cap. Westmarches set this to math.MaxInt32 (effectively unlimited) and gate per-session admission via SessionCapacity instead.
	SessionCapacity int      `bun:",notnull,default:6" json:"session_capacity"` // Westmarch tripwire — seats per sitting. Joining past this admits anyway and DMs the campaign DM. Ignored for non-westmarch campaigns.
	IsOpen          bool     `bun:",notnull,default:false" json:"is_open"`
	IsOneshot       bool     `bun:",notnull,default:false" json:"is_oneshot"`
	IsWestmarch     bool     `bun:",notnull,default:false" json:"is_westmarch"`
	Warnings        []string `bun:",array,type:jsonb" json:"warnings,omitempty"`
	Extra           string   `bun:",default:''" json:"extra,omitempty"`

	Schedule CampaignSchedule `bun:"embed:"` // Campaign schedule.

	// Links and media
	Links          []string `bun:",array,type:jsonb" json:"links,omitempty"`
	VTTLink        string   `bun:",default:''" json:"vtt_link,omitempty"`
	PlayerSheetURL string   `bun:",default:''" json:"player_sheet_url,omitempty"`
	CampaignMedia  []string `bun:",array,type:jsonb" json:"campaign_media,omitempty"`

	CampaignPlayers []CampaignPlayer `bun:"rel:has-many,join:id=campaign_id" json:"campaign_players,omitempty"`
	Sessions        []Session        `bun:"rel:has-many,join:id=campaign_id" json:"sessions,omitempty"`

	// Can you add a new player *even if* the Campaign is full?
	CanOverflow bool `bun:",notnull,default:false" json:"can_overflow"`

	// Discord role ID that grants access to this campaign's channel.
	// Empty string means no role has been linked yet.
	RoleID string `bun:",default:''" json:"role_id"`

	// Discord channel/thread IDs created on approval.
	ChannelID             string `bun:",default:''" json:"channel_id"`
	CategoryID            string `bun:",default:''" json:"category_id"`
	AnnouncementsThreadID string `bun:",default:''" json:"announcements_thread_id"`

	// Has this campaign been approved to be published?
	IsApproved bool `bun:",notnull,default:false"`

	// Archival: an archived campaign is an immutable record. No mutations allowed.
	// Archival happens when a DM explicitly abandons the campaign or leaves the server.
	IsArchived     bool      `bun:",notnull,default:false"`
	ArchivedAt     time.Time `bun:",nullzero"`
	ArchivedReason string    `bun:",nullzero"`

	// Soft delete: staff-triggered. Row is retained for the admin historical log.
	// bun automatically adds WHERE deleted_at IS NULL to all normal queries.
	DeletedAt time.Time `bun:",soft_delete,nullzero"`
}

// CanMutate returns false if the campaign is archived (immutable).
func (c *Campaign) CanMutate() bool {
	return !c.IsArchived
}

/*
DisplaySlots renders the roster cap for player-facing UI.

Westmarches store math.MaxInt32 to mean "no roster cap" and must never leak that number;
legacy rows with Slots <= 0 mean unset/unlimited.

The "10+" form is similar to the stock counter at a dept-store: hide the exact figure once it stops being meaningful.
*/
func (c *Campaign) DisplaySlots() string {
	if c.Slots <= 0 {
		return "Unlimited"
	}
	if c.Slots > 10 {
		return "10+"
	}
	return fmt.Sprintf("%d", c.Slots)
}

/*
PlayerMap returns a map of player ID to CampaignPlayer for quick lookups.

Requires CampaignPlayers to be loaded (via Relation).
*/
func (c *Campaign) PlayerMap() map[string]*CampaignPlayer {
	m := make(map[string]*CampaignPlayer, len(c.CampaignPlayers))
	for i := range c.CampaignPlayers {
		m[c.CampaignPlayers[i].PlayerID] = &c.CampaignPlayers[i]
	}
	return m
}

/*
DMMap returns a map of player ID to CampaignPlayer for DMs only.

Requires CampaignPlayers to be loaded (via Relation).
*/
func (c *Campaign) DMMap() map[string]*CampaignPlayer {
	m := make(map[string]*CampaignPlayer)
	for i := range c.CampaignPlayers {
		if c.CampaignPlayers[i].Role == RoleDM {
			m[c.CampaignPlayers[i].PlayerID] = &c.CampaignPlayers[i]
		}
	}
	return m
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
	Frequency     CampaignFrequency `bun:",notnull,default:'weekly'" json:"frequency"`
	DayOfWeek     int               `bun:",notnull,default:-1" json:"day_of_week"`   // 0=Mon..6=Sun, -1=unset
	StartTime     string            `bun:",notnull,default:''" json:"start_time"`    // "HH:MM" in UTC, empty=unset
	DurationHours float64           `bun:",notnull,default:3" json:"duration_hours"` // session length in hours, default 3h
	CreatedAt     time.Time         `bun:",notnull,default:current_timestamp" json:"created_at"`
	LastSession   time.Time         `bun:",nullzero" json:"last_session"`
	NextSession   time.Time         `bun:",nullzero" json:"next_session"`
	AlertSent     bool              `bun:",notnull,default:false" json:"alert_sent"`
}

// DayName returns the display name for the schedule's day of week.
func (s CampaignSchedule) DayName() string {
	days := [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
		return "Unset"
	}
	return days[s.DayOfWeek]
}

// HasSchedule returns true if the campaign has a day and time set.
func (s CampaignSchedule) HasSchedule() bool {
	return s.DayOfWeek >= 0 && s.StartTime != ""
}

/*
	NormalizeTag converts a campaign name into a URL-safe tag.

"Curse of Strahd" -> "curse-of-strahd"
*/
func NormalizeTag(name string) string {
	if idx := strings.IndexByte(name, ':'); idx != -1 {
		name = name[:idx]
	}
	tag := strings.ToLower(strings.TrimSpace(name))

	// Replace non-alphanumeric characters with hyphens; allow unicode letters (e.g. ñ, é).
	var b strings.Builder
	prevHyphen := false
	for _, r := range tag {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}

	tag = strings.Trim(b.String(), "-")

	if len(tag) > 30 {
		tag = tag[:30]
		tag = strings.TrimRight(tag, "-")
	}

	return tag
}

// UniqueTag returns base if no campaign owns it; otherwise appends -2, -3, … until one is free.
func UniqueTag(database *bun.DB, base string) (string, error) {
	if base == "" {
		base = "campaign"
	}
	candidate := base
	ctx := context.Background()
	for n := 2; n < 1000; n++ {
		exists, err := database.NewSelect().Model((*Campaign)(nil)).Where("tag = ?", candidate).Exists(ctx)
		if err != nil {
			return "", fmt.Errorf("checking tag %q: %w", candidate, err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return "", fmt.Errorf("could not find a unique tag for %q", base)
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
	Westmarch CampaignFrequency = "westmarch"
	Irregular CampaignFrequency = "irregular"
)

func (c *Campaign) CreateCampaign(
	db *bun.DB,
	dmID string,
	playerIDs []string,
	name string,
	tag string,
	description string,
	conf *GameConfig,
	slots int,
	isOpen bool,
	isOneshot bool,
	warnings []string,
	extraInfo string,
	schedule *CampaignSchedule,
	links []string,
	vtt string,
	books []string,
	otherGameInfo []string,
	isApproved bool,
) (*Campaign, error) {
	ctx := context.Background()

	// get DM from DB
	var dm Player
	err := db.NewSelect().Model(&dm).Where("id = ?", dmID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	// get players from DB
	var players []Player
	if len(playerIDs) > 0 {
		err = db.NewSelect().Model(&players).Where("id IN (?)", bun.In(playerIDs)).Scan(ctx)
		if err != nil {
			return nil, err
		}
	}

	// create campaign entry on DB
	campaign := &Campaign{
		ID:            uuid.NewString(),
		Name:          name,
		Tag:           tag,
		DungeonMaster: dmID,
		DM:            &dm,
		Description:   description,
		Game:          *conf,
		Slots:         slots,
		IsOpen:        isOpen,
		IsOneshot:     isOneshot,
		Warnings:      warnings,
		Extra:         extraInfo,
		Schedule:      *schedule,
		Links:         links,
		VTTLink:       vtt,
		IsApproved:    false,
	}

	_, err = db.NewInsert().Model(campaign).Exec(ctx)
	if err != nil {
		return nil, err
	}

	// create CampaignPlayer entries for each player
	for _, p := range players {
		cp := &CampaignPlayer{
			PlayerID:   p.ID,
			CampaignID: campaign.ID,
			Role:       RolePlayer,
			Status:     StatusActive,
		}
		_, err = db.NewInsert().Model(cp).Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	// also add the DM as a CampaignPlayer
	// dm's are players too :)
	dmEntry := &CampaignPlayer{
		PlayerID:   dmID,
		CampaignID: campaign.ID,
		Role:       RoleDM,
		Status:     StatusActive,
	}
	_, err = db.NewInsert().Model(dmEntry).Exec(ctx)
	if err != nil {
		return nil, err
	}

	if campaign.IsWestmarch {
		campaign.Slots = math.MaxInt32
	}

	return campaign, nil
}

// RefreshNextSession updates Campaign.Schedule.NextSession to the earliest upcoming session
// in the sessions table. Should be called after inserting or deleting a Session.
func (c *Campaign) RefreshNextSession(db *bun.DB) error {
	ctx := context.Background()
	var earliest Session
	err := db.NewSelect().Model(&earliest).
		Where("campaign_id = ? AND status = ? AND scheduled_at > ?", c.ID, SessionUpcoming, time.Now().UTC()).
		OrderExpr("scheduled_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		// No upcoming sessions — clear NextSession.
		c.Schedule.NextSession = time.Time{}
		c.Schedule.AlertSent = false
	} else {
		c.Schedule.NextSession = earliest.ScheduledAt
		c.Schedule.AlertSent = earliest.AlertSent
	}
	_, err2 := db.NewUpdate().Model(c).
		Column("next_session", "alert_sent").
		Where("id = ?", c.ID).
		Exec(ctx)
	return err2
}
