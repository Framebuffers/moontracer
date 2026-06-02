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

	/*
		CampaignsCategoryID is the Discord category where new campaign channels are created on approval.
		Empty means SetupNewChannel will find-or-create "Campaigns" by name.
	*/
	CampaignsCategoryID string `bun:",default:''" json:"campaigns_category_id"`

	/*
		ArchivedCategoryID is the Discord category that retired campaign channels are moved to on
		archive or deletion. Empty means channels are not moved.
	*/
	ArchivedCategoryID string `bun:",default:''" json:"archived_category_id"`

	/*
		BillboardCategoryID is the Discord category channel that contains all billboard
		forum channels.

		Empty means PostBillboard will find-or-create "Campaigns" by name.
	*/
	BillboardCategoryID string `bun:",default:''" json:"billboard_category_id"`

	/*
		CampaignChannelID is a general-purpose text channel for campaign-related announcements.
		Empty means the feature is disabled.
	*/
	CampaignChannelID string `bun:",default:''" json:"campaign_channel_id"`
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
