package cdn

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

/*
Package cdn handles campaign cover images hosted on the Discord CDN.

Approach:
	We store the CDN URL as-is (CoverCachedURL).

	Discord signs these URLs with a sliding expiry, but for the vast majority of campaigns the URL
	stays valid long enough to be useful.

	When a URL starts 404'ing (source deleted, token rotation we can't refresh), we clear the ref and the UI
	falls back to text-only. Then, users can re-upload.

The 3 ref columns on Campaign (CoverChannelID/CoverMessageID/CoverAttachmentID)
are reserved for a future storage-channel approach if URL churn becomes a
real problem. They're not populated as of now.
*/

const headTimeout = 3 * time.Second

/*
ResolveCoverURL returns the campaign's cover URL if one is set and still reachable.

A short HEAD request verifies liveness; on 404 we clear the ref. Any other error is non-fatal and returns the URL anyway (optimistaclly).
*/
func ResolveCoverURL(ctx context.Context, db *bun.DB, c *models.Campaign) string {
	if c.CoverCachedURL == "" {
		return ""
	}

	client := &http.Client{Timeout: headTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.CoverCachedURL, nil)
	if err != nil {
		return c.CoverCachedURL
	}
	resp, err := client.Do(req)
	if err != nil {
		return c.CoverCachedURL
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		clearCover(ctx, db, c)
		log.Printf("cdn: cover for campaign %s 404'd, cleared ref", c.ID)
		return ""
	}

	return c.CoverCachedURL
}

/*
SetCover records a new cover URL on the campaign.

Overwrites any prior value and stamps the refresh time.
*/
func SetCover(ctx context.Context, db *bun.DB, c *models.Campaign, url string) error {
	now := time.Now()
	c.CoverCachedURL = url
	c.CoverCachedRefreshed = &now
	_, err := db.NewUpdate().Model(c).
		Column("cover_cached_url", "cover_cached_refreshed").
		WherePK().Exec(ctx)
	return err
}

func clearCover(ctx context.Context, db *bun.DB, c *models.Campaign) {
	c.CoverCachedURL = ""
	c.CoverCachedRefreshed = nil
	if _, err := db.NewUpdate().Model(c).
		Column("cover_cached_url", "cover_cached_refreshed").
		WherePK().Exec(ctx); err != nil {
		log.Printf("cdn: failed to clear cover for %s: %v", c.ID, err)
	}
}
