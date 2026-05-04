package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/testutil"
)

func seedMedia(t *testing.T, db *bun.DB, id, ownerID, campaignID, path string, kind models.MediaKind) *models.Media {
	t.Helper()
	m := &models.Media{
		ID:         id,
		OwnerID:    ownerID,
		CampaignID: campaignID,
		Path:       path,
		Kind:       kind,
		Name:       id,
		CreatedAt:  time.Now(),
	}
	_, err := db.NewInsert().Model(m).Exec(context.Background())
	require.NoError(t, err)
	return m
}

/*
MediaByOwner returns all records for a given owner.

When:

	Two records share an owner; a third belongs to someone else.

Expected:

	Only the two matching records are returned.
*/
func TestMediaByOwner(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "", "/img/a.webp", models.KindCoverArt)
	seedMedia(t, database, "m2", "owner1", "", "/img/b.webp", models.KindTokenPlayer)
	seedMedia(t, database, "m3", "owner2", "", "/img/c.webp", models.KindCoverArt)

	results, err := models.MediaByOwner(database, "owner1")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}
	assert.True(t, ids["m1"])
	assert.True(t, ids["m2"])
	assert.False(t, ids["m3"])
}

/*
MediaByOwner returns empty when owner has no records.

When:

	Database has records belonging to a different owner.

Expected:

	Empty result, no error.
*/
func TestMediaByOwner_Empty(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "", "/img/a.webp", models.KindCoverArt)

	results, err := models.MediaByOwner(database, "owner2")
	require.NoError(t, err)
	assert.Empty(t, results)
}

/*
MediaByCampaign returns only records matching both campaign and kind.

When:

	Campaign has one cover art and one player token; another campaign has its own cover art.

Expected:

	Only the cover art for the requested campaign is returned.
*/
func TestMediaByCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "camp1", "/img/cover.webp", models.KindCoverArt)
	seedMedia(t, database, "m2", "owner1", "camp1", "/img/token.webp", models.KindTokenPlayer)
	seedMedia(t, database, "m3", "owner2", "camp2", "/img/other.webp", models.KindCoverArt)

	results, err := models.MediaByCampaign(database, "camp1", models.KindCoverArt)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "m1", results[0].ID)
}

/*
MediaByCampaign returns empty when no records match kind.

When:

	Campaign has records, but none of the requested kind.

Expected:

	Empty result, no error.
*/
func TestMediaByCampaign_WrongKind(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "camp1", "/img/token.webp", models.KindTokenPlayer)

	results, err := models.MediaByCampaign(database, "camp1", models.KindCoverArt)
	require.NoError(t, err)
	assert.Empty(t, results)
}

/*
MediaByKind returns all records of a given kind across owners.

When:

	Two cover art records exist alongside one player token.

Expected:

	Only the two cover art records are returned.
*/
func TestMediaByKind(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "camp1", "/img/a.webp", models.KindCoverArt)
	seedMedia(t, database, "m2", "owner2", "camp2", "/img/b.webp", models.KindCoverArt)
	seedMedia(t, database, "m3", "owner1", "",      "/img/tok.webp", models.KindTokenPlayer)

	results, err := models.MediaByKind(database, models.KindCoverArt)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}
	assert.True(t, ids["m1"])
	assert.True(t, ids["m2"])
	assert.False(t, ids["m3"])
}

/*
MediaByPath returns the record at the given path.

When:

	Two records exist with different paths.

Expected:

	The record matching the path is returned.
*/
func TestMediaByPath(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "", "/img/cover.webp", models.KindCoverArt)
	seedMedia(t, database, "m2", "owner1", "", "/img/token.webp", models.KindTokenPlayer)

	result, err := models.MediaByPath(database, "/img/cover.webp")
	require.NoError(t, err)
	assert.Equal(t, "m1", result.ID)
	assert.Equal(t, models.KindCoverArt, result.Kind)
}

/*
MediaByPath returns an error for a path that does not exist.

When:

	Database has records, but none match the requested path.

Expected:

	Error is returned.
*/
func TestMediaByPath_NotFound(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedMedia(t, database, "m1", "owner1", "", "/img/cover.webp", models.KindCoverArt)

	_, err := models.MediaByPath(database, "/img/missing.webp")
	assert.Error(t, err)
}
