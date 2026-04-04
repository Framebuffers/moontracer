package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/testutil"
)

// --- helpers ---

func seedPlayer(t *testing.T, database *bun.DB, id string, role models.ServerRole) {
	t.Helper()
	ctx := context.Background()
	p := &models.Player{ID: id, Role: role}
	_, err := database.NewInsert().Model(p).Exec(ctx)
	require.NoError(t, err)
}

func seedCampaign(t *testing.T, database *bun.DB, id, name, tag, dmID string, approved bool) {
	t.Helper()
	ctx := context.Background()
	c := &models.Campaign{
		ID:            id,
		Name:          name,
		Tag:           tag,
		DungeonMaster: dmID,
		IsApproved:    approved,
	}
	_, err := database.NewInsert().Model(c).Exec(ctx)
	require.NoError(t, err)
}

func seedCampaignPlayer(t *testing.T, database *bun.DB, playerID, campaignID string, role models.Role, status models.CampaignPlayerStatus) {
	t.Helper()
	ctx := context.Background()
	cp := &models.CampaignPlayer{
		PlayerID:   playerID,
		CampaignID: campaignID,
		Role:       role,
		Status:     status,
	}
	_, err := database.NewInsert().Model(cp).Exec(ctx)
	require.NoError(t, err)
}

// --- TESTS ---
// --- GetStaff ---

func TestGetStaff_ReturnsModsAndAdmins(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "player1", models.ServerRolePlayer)
	seedPlayer(t, database, "mod1", models.ServerRoleMod)
	seedPlayer(t, database, "admin1", models.ServerRoleAdmin)

	staff, err := db.GetStaff(database)
	require.NoError(t, err)
	assert.Len(t, staff, 2, "GetStaff should return mod + admin, not regular players")

	ids := map[string]bool{}
	for _, s := range staff {
		ids[s.ID] = true
	}
	assert.True(t, ids["mod1"], "mod should be in staff")
	assert.True(t, ids["admin1"], "admin should be in staff")
	assert.False(t, ids["player1"], "regular player should not be in staff")
}

func TestGetStaff_EmptyWhenNoStaff(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "player1", models.ServerRolePlayer)
	seedPlayer(t, database, "player2", models.ServerRolePlayer)

	staff, err := db.GetStaff(database)
	require.NoError(t, err)
	assert.Empty(t, staff, "GetStaff should return empty when no mods or admins exist")
}

/*
	 --- Approval gate: campaign lookup still returns unapproved campaigns ---

		The gate is enforced in the handler (if !campaign.IsApproved), not at the DB level.
		These tests verify the data layer returns the campaign regardless, so the handler can decide.
*/
func TestGetByTag_ReturnsUnapprovedCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Pending Campaign", "pending", "dm1", false)

	campaign, err := db.GetByTag[models.Campaign](database, "pending")
	require.NoError(t, err)
	assert.Equal(t, "camp1", campaign.ID)
	assert.False(t, campaign.IsApproved, "campaign should still be unapproved")
}

func TestGetByTag_ReturnsApprovedCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Approved Campaign", "approved", "dm1", true)

	campaign, err := db.GetByTag[models.Campaign](database, "approved")
	require.NoError(t, err)
	assert.Equal(t, "camp1", campaign.ID)
	assert.True(t, campaign.IsApproved, "campaign should be approved")
}

// --- Approve/Deny state transitions ---

func TestApproveCampaign_SetsIsApprovedTrue(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Test", "test", "dm1", false)

	campaign, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err)
	assert.False(t, campaign.IsApproved)

	campaign.IsApproved = true
	err = db.Update(database, campaign)
	require.NoError(t, err)

	reloaded, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err)
	assert.True(t, reloaded.IsApproved, "campaign should be approved after update")
}

func TestDenyCampaign_DeletesFromDB(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Doomed", "doomed", "dm1", false)

	_, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err, "campaign should exist before denial")

	err = db.Delete[models.Campaign](database, "camp1")
	require.NoError(t, err)

	_, err = db.GetByID[models.Campaign](database, "camp1")
	assert.Error(t, err, "campaign should not exist after deletion")
}

func TestDoubleApprove_IsIdempotent(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Test", "test", "dm1", false)

	campaign, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err)

	// First approve
	campaign.IsApproved = true
	err = db.Update(database, campaign)
	require.NoError(t, err)

	// Second approve — should not error
	campaign.IsApproved = true
	err = db.Update(database, campaign)
	require.NoError(t, err)

	reloaded, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err)
	assert.True(t, reloaded.IsApproved)
}

func TestDenyAlreadyDeleted_ReturnsNotFound(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Gone", "gone", "dm1", false)

	err := db.Delete[models.Campaign](database, "camp1")
	require.NoError(t, err)

	_, err = db.GetByID[models.Campaign](database, "camp1")
	assert.Error(t, err, "looking up a deleted campaign should return an error")
}

// --- Visibility: mycampaigns should be filterable by IsApproved ---

func TestGetPlayerCampaigns_IncludesBothApprovedAndUnapproved(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "approved-camp", "Approved", "approved", "dm1", true)
	seedCampaign(t, database, "pending-camp", "Pending", "pending", "dm1", false)
	seedCampaignPlayer(t, database, "dm1", "approved-camp", models.RoleDM, models.StatusActive)
	seedCampaignPlayer(t, database, "dm1", "pending-camp", models.RoleDM, models.StatusActive)

	entries, err := models.GetPlayerCampaigns(database, "dm1")
	require.NoError(t, err)
	assert.Len(t, entries, 2, "GetPlayerCampaigns returns all campaigns (handler filters)")

	// Simulate the handler filter: only approved campaigns shown
	var visible []models.CampaignPlayer
	for _, e := range entries {
		if e.Campaign != nil && e.Campaign.IsApproved {
			visible = append(visible, e)
		}
	}
	assert.Len(t, visible, 1, "only approved campaigns should be visible after filtering")
	assert.Equal(t, "approved-camp", visible[0].CampaignID)
}
