package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/framebuffers/moontracer/internal/manager/models"
)

/*
newTestDB sets up an in-memory bun DB with the full schema migrated.

Development Note:

	Rather than reusing testutil.NewTestDB, this method is inlined here
	because testutil imports this package, which would create a test-import cycle.
*/
func newTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())
	if err := Migrate(bunDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() { bunDB.Close() })
	return bunDB
}

/*
Unit Testing
ScrubOrphanedCampaignPlayers- guild-DB sanity sweep

Covers the cleanup helper that runs on every guild-DB open.
The function is idempotent and safe to call when no orphans exist.

Development Note:

	Older code paths (pre commit c32d817) deleted campaigns without cascading their
	CampaignPlayer rows, leaving orphan memberships that auth.go then walked into.

	The scrub deletes any CampaignPlayer whose campaign_id has no matching Campaign row.

*/

const (
	scrubLiveCampID    = "live-1"
	scrubOrphanCampID  = "orphan-1"
	scrubOrphanCampID2 = "orphan-2"
	scrubMember1ID     = "member-1"
	scrubMember2ID     = "member-2"
	scrubGhost1ID      = "ghost-1"
	scrubGhost2ID      = "ghost-2"
	scrubGhost3ID      = "ghost-3"
)

/*
Removes orphans, keeps live memberships.

When:

	A guild DB has 2 CampaignPlayer rows tied to a real Campaign and 3 rows
	pointing at nonexistent campaign IDs.

Expected:

	The 3 orphans are deleted (RowsAffected reports 3); the 2 live
	memberships survive untouched.
*/
func TestScrubOrphans_RemovesOnlyDanglingRows(t *testing.T) {
	database := newTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Campaign{
		ID:            scrubLiveCampID,
		Name:          "Live",
		Tag:           "live",
		DungeonMaster: scrubMember1ID,
		IsApproved:    true,
	}).Exec(ctx)
	require.NoError(t, err)

	live := []models.CampaignPlayer{
		{PlayerID: scrubMember1ID, CampaignID: scrubLiveCampID, Role: models.RoleDM, Status: models.StatusActive},
		{PlayerID: scrubMember2ID, CampaignID: scrubLiveCampID, Role: models.RolePlayer, Status: models.StatusActive},
	}
	for i := range live {
		_, err := database.NewInsert().Model(&live[i]).Exec(ctx)
		require.NoError(t, err)
	}

	orphans := []models.CampaignPlayer{
		{PlayerID: scrubGhost1ID, CampaignID: scrubOrphanCampID, Role: models.RolePlayer, Status: models.StatusActive},
		{PlayerID: scrubGhost2ID, CampaignID: scrubOrphanCampID, Role: models.RolePlayer, Status: models.StatusFinished},
		{PlayerID: scrubGhost3ID, CampaignID: scrubOrphanCampID2, Role: models.RoleDM, Status: models.StatusActive},
	}
	for i := range orphans {
		_, err := database.NewInsert().Model(&orphans[i]).Exec(ctx)
		require.NoError(t, err)
	}

	n, err := ScrubOrphanedCampaignPlayers(database)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n, "all 3 orphan rows should be deleted")

	survivors, err := models.GetCampaignPlayers(database, scrubLiveCampID)
	require.NoError(t, err)
	assert.Len(t, survivors, 2, "live memberships must survive the scrub")

	var remaining []models.CampaignPlayer
	require.NoError(t, database.NewSelect().Model(&remaining).
		Where("campaign_id IN (?, ?)", scrubOrphanCampID, scrubOrphanCampID2).
		Scan(ctx))
	assert.Empty(t, remaining, "no orphans should be left behind")
}

/*
No-op when nothing is orphaned.

When:

	Every CampaignPlayer row points at a real Campaign.

Expected:

	RowsAffected reports 0; no error. Confirms the scrub is safe to invoke
	on every guild-DB open without touching healthy state.
*/
func TestScrubOrphans_NoOpWhenClean(t *testing.T) {
	database := newTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Campaign{
		ID:            scrubLiveCampID,
		Name:          "Live",
		Tag:           "live",
		DungeonMaster: scrubMember1ID,
		IsApproved:    true,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID: scrubMember1ID, CampaignID: scrubLiveCampID,
		Role: models.RoleDM, Status: models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	n, err := ScrubOrphanedCampaignPlayers(database)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "clean DB should report zero scrubbed rows")
}

/*
Empty DB is safe.

When:

	The DB has no campaigns and no campaign_player rows.

Expected:

	RowsAffected = 0, no error. Catches the freshly-migrated guild path.
*/
func TestScrubOrphans_EmptyDB(t *testing.T) {
	database := newTestDB(t)

	n, err := ScrubOrphanedCampaignPlayers(database)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}
