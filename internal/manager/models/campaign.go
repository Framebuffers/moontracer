package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Campaign represents a DnD campaign.
type Campaign struct {
	bun.BaseModel `bun:"table:campaigns"`

	// Server-generated unique identifier. This is the ID that will be used throughout to identify this particular campaign.
	ID string `bun:",pk,notnull" json:"id"`

	// Display name for the campaign.
	Name string `bun:",notnull" json:"name"`

	// Short, user-facing identifier for lookups (e.g. "strahd", "avalon", "itzaal", "nuevosur").
	Tag string `bun:",unique,notnull" json:"tag"`

	// Player FK of the user (DM) who created the campaign.
	DungeonMaster string  `bun:",notnull" json:"dm"`
	DM            *Player `bun:"rel:belongs-to,join:dungeon_master=id" json:"dm_player,omitempty"`

	// Campaign's description, synopsis, details, lore, or whatever you want to add to describe your setting.
	Description string `bun:",notnull,default:''" json:"description"`

	// Embedded singular game configuration. Stored as flat columns.
	Game GameConfig `bun:"embed:"`

	// Details about your campaign, like open slots, the style, trigger warnings, extra info by the DM to be added to the Campaign's description.
	Slots       int      `bun:",notnull,default:0" json:"slots"`
	IsOpen      bool     `bun:",notnull,default:false" json:"is_open"`
	IsOneshot   bool     `bun:",notnull,default:false" json:"is_oneshot"`
	IsWestmarch bool     `bun:",notnull,default:false" json:"is_westmarch"`
	Warnings    []string `bun:",array,type:jsonb" json:"warnings,omitempty"`
	Extra       string   `bun:",default:''" json:"extra,omitempty"`

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
	NextSession time.Time         `bun:",nullzero" json:"next_session"`
	AlertSent   bool              `bun:",notnull,default:false" json:"alert_sent"`
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

	return campaign, nil
}
