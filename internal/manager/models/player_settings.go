package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

/*
PlayerSettings holds per-player preferences, primarily notification toggles.

One row per Player, keyed by PlayerID. Defaults are all-on so players get
everything until they opt out.
*/
type PlayerSettings struct {
	bun.BaseModel `bun:"table:player_settings"`

	PlayerID            string  `bun:",pk,notnull" json:"player_id"`
	Player              *Player `bun:"rel:belongs-to,join:player_id=id" json:"player,omitempty"`
	NotifyAnnouncements bool    `bun:",notnull,default:true" json:"notify_announcements"`
	NotifySessionRemind bool    `bun:",notnull,default:true" json:"notify_session_remind"`
	NotifyInvitations   bool    `bun:",notnull,default:true" json:"notify_invitations"`
	Timezone            string  `bun:",notnull,default:'UTC'" json:"timezone"`
}

// Location returns the player's preferred *time.Location, falling back to UTC on any error.
func (s *PlayerSettings) Location() *time.Location {
	if s.Timezone == "" || s.Timezone == "UTC" {
		return time.UTC
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// GetOrCreatePlayerSettings loads settings for a player, creating defaults if none exist.
func GetOrCreatePlayerSettings(db *bun.DB, playerID string) (*PlayerSettings, error) {
	ctx := context.Background()
	var s PlayerSettings
	err := db.NewSelect().Model(&s).Where("player_id = ?", playerID).Scan(ctx)
	if err == nil {
		return &s, nil
	}

	s = PlayerSettings{
		PlayerID:            playerID,
		NotifyAnnouncements: true,
		NotifySessionRemind: true,
		NotifyInvitations:   true,
		Timezone:            "UTC",
	}
	if _, err := db.NewInsert().Model(&s).Exec(ctx); err != nil {
		return nil, err
	}
	return &s, nil
}
