package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type MediaKind string

const (
	KindCoverArt      MediaKind = "cover_art"
	KindTokenPlayer   MediaKind = "token_player"
	KindTokenCampaign MediaKind = "token_campaign"
	KindVTTLink       MediaKind = "vtt_link"
	KindInternal      MediaKind = "internal"
)

/*
Media is the single record for any externally-stored asset:
cover art, player/NPC tokens, VTT links, etc.

Path holds a local disk path (for files) or a raw URL (for links).
URL holds the public CDN URL served to Discord (empty for non-file assets).
*/
type Media struct {
	bun.BaseModel `bun:"table:media"`

	ID         string    `bun:",pk,notnull"  json:"id"`
	OwnerID    string    `bun:",notnull"     json:"owner_id"`
	CampaignID string    `bun:",nullzero"    json:"campaign_id,omitempty"`
	Path       string    `bun:",notnull"     json:"path"`
	URL        string    `bun:",nullzero"    json:"url,omitempty"`
	Kind       MediaKind `bun:",notnull"     json:"kind"`
	Name       string    `bun:",notnull"     json:"name"`
	MimeType   string    `bun:",nullzero"    json:"mime_type,omitempty"`
	CreatedAt  time.Time `bun:",notnull"     json:"created_at"`
}

func MediaByOwner(db *bun.DB, ownerID string) ([]*Media, error) {
	var out []*Media
	err := db.NewSelect().Model(&out).Where("owner_id = ?", ownerID).OrderExpr("created_at DESC").Scan(context.Background())
	return out, err
}

func MediaByCampaign(db *bun.DB, campaignID string, kind MediaKind) ([]*Media, error) {
	var out []*Media
	err := db.NewSelect().Model(&out).
		Where("campaign_id = ?", campaignID).
		Where("kind = ?", kind).
		OrderExpr("created_at DESC").
		Scan(context.Background())
	return out, err
}

func MediaByKind(db *bun.DB, kind MediaKind) ([]*Media, error) {
	var out []*Media
	err := db.NewSelect().Model(&out).Where("kind = ?", kind).OrderExpr("created_at DESC").Scan(context.Background())
	return out, err
}

func MediaByPath(db *bun.DB, path string) (*Media, error) {
	var out Media
	err := db.NewSelect().Model(&out).Where("path = ?", path).Limit(1).Scan(context.Background())
	return &out, err
}

/*
CoverURLForCampaign returns the public CDN URL for the most-recent cover art
uploaded for the campaign, or an empty string if none exists.
*/
func CoverURLForCampaign(db *bun.DB, campaignID string) string {
	media, err := MediaByCampaign(db, campaignID, KindCoverArt)
	if err != nil || len(media) == 0 {
		return ""
	}
	return media[0].URL
}
