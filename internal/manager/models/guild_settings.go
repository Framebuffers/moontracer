package models

import (
	"context"

	"github.com/uptrace/bun"
)

// GuildSettings holds per-guild configuration. One row per guild database.
type GuildSettings struct {
	bun.BaseModel `bun:"table:guild_settings"`

	ID int `bun:",pk,autoincrement"`

	/*
		Billboard forum channel IDs for each campaign format.
		Empty string means "not configured"; PostBillboard will find-or-create.
	*/
	BillboardChannelCampaign  string `bun:",default:''" json:"billboard_channel_campaign"`
	BillboardChannelOneshot   string `bun:",default:''" json:"billboard_channel_oneshot"`
	BillboardChannelWestmarch string `bun:",default:''" json:"billboard_channel_westmarch"`
}

// GetOrCreateGuildSettings returns the single GuildSettings row, inserting defaults if absent.
func GetOrCreateGuildSettings(db *bun.DB) (*GuildSettings, error) {
	ctx := context.Background()
	s := &GuildSettings{}
	err := db.NewSelect().Model(s).Limit(1).Scan(ctx)
	if err == nil {
		return s, nil
	}
	s = &GuildSettings{ID: 1}
	_, err = db.NewInsert().Model(s).On("CONFLICT (id) DO NOTHING").Exec(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}
