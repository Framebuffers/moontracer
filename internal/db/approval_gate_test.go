package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/testutil"
)

/*
Helper functions
*/

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

/*
Unit Testing: staff queries and approval gate.
*/

/*
GetStaff returns mods and admins.

When:

	Database has one player, one mod, one admin.

Expected:

	Only the mod and admin are returned. Regular players are excluded.
*/
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

/*
GetStaff returns empty when no staff exist.

When:

	Database has only regular players, no mods or admins.

Expected:

	Empty result.
*/
func TestGetStaff_EmptyWhenNoStaff(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "player1", models.ServerRolePlayer)
	seedPlayer(t, database, "player2", models.ServerRolePlayer)

	staff, err := db.GetStaff(database)
	require.NoError(t, err)
	assert.Empty(t, staff, "GetStaff should return empty when no mods or admins exist")
}

/*
Approval gate: campaign lookup returns unapproved campaigns.

	The gate is enforced in the handler (if !campaign.IsApproved), not at the DB level.
	These tests verify the data layer returns the campaign regardless, so the handler can decide.
*/

/*
GetByTag returns an unapproved campaign.

When:

	Campaign exists with IsApproved = false.

Expected:

	Campaign is returned. The data layer does not filter by approval status.
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

/*
GetByTag returns an approved campaign.

When:

	Campaign exists with IsApproved = true.

Expected:

	Campaign is returned with approval flag intact.
*/
func TestGetByTag_ReturnsApprovedCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Approved Campaign", "approved", "dm1", true)

	campaign, err := db.GetByTag[models.Campaign](database, "approved")
	require.NoError(t, err)
	assert.Equal(t, "camp1", campaign.ID)
	assert.True(t, campaign.IsApproved, "campaign should be approved")
}

/*
Approve campaign sets IsApproved to true.

When:

	Campaign starts unapproved, then IsApproved is set to true and updated.

Expected:

	Reloaded campaign has IsApproved = true.
*/
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

/*
Deny campaign deletes from database.

When:

	Campaign exists, then is deleted (denial = deletion).

Expected:

	Lookup after deletion returns an error.
*/
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

/*
Double approve is idempotent.

When:

	Campaign is approved twice in succession.

Expected:

	No error on second approve. Campaign remains approved.
*/
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

	// Second approve - should not error
	campaign.IsApproved = true
	err = db.Update(database, campaign)
	require.NoError(t, err)

	reloaded, err := db.GetByID[models.Campaign](database, "camp1")
	require.NoError(t, err)
	assert.True(t, reloaded.IsApproved)
}

/*
Deny already-deleted campaign returns not found.

When:

	Campaign is deleted, then looked up again.

Expected:

	Lookup returns an error. No phantom records.
*/
func TestDenyAlreadyDeleted_ReturnsNotFound(t *testing.T) {
	database := testutil.NewTestDB(t)

	seedPlayer(t, database, "dm1", models.ServerRolePlayer)
	seedCampaign(t, database, "camp1", "Gone", "gone", "dm1", false)

	err := db.Delete[models.Campaign](database, "camp1")
	require.NoError(t, err)

	_, err = db.GetByID[models.Campaign](database, "camp1")
	assert.Error(t, err, "looking up a deleted campaign should return an error")
}

/*
GetPlayerCampaigns includes both approved and unapproved.

When:

	DM owns one approved and one unapproved campaign.

Expected:

	GetPlayerCampaigns returns both. Handler-level filtering narrows to approved only.
*/
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
