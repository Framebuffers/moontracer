package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/testutil"
)

/*
	Unit Testing: enforce authorization and sovereignty rules while on Debug Mode
*/

/*
Helper functions
*/
func withDebugAdmin(t *testing.T, id string, fn func()) {
	t.Helper()
	origSafe := guard.SafeMode
	origID := guard.DebugAdminID
	guard.SafeMode = true
	guard.DebugAdminID = id
	t.Cleanup(func() {
		guard.SafeMode = origSafe
		guard.DebugAdminID = origID
	})
	fn()
}

/*
Test
syncRole

Flow:
 1. Register a plain player.
 2. Sync without Discord admins.
    a) Debug Admin should still be elevated.
*/
func TestSyncRoles_DebugAdminElevated(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	withDebugAdmin(t, "debug1", func() {
		err := syncRoles(database, nil, nil)
		require.NoError(t, err)

		var p models.Player
		err = database.NewSelect().Model(&p).Where("id = ?", "debug1").Scan(ctx)
		require.NoError(t, err)
		assert.Equal(t, models.ServerRoleAdmin, p.Role, "debug admin should be elevated to admin after sync")
	})
}

/*
When:

	SafeMode = OFF;
	DebugAdminID = set;
	User does NOT have the Discord admin role.

Expected:

	Elevation denied. In production, DEBUG_ADMIN_ID requires the Discord role as confirmation.
*/
func TestSyncRoles_DebugAdminDeniedWithoutDiscordRole(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	origSafe := guard.SafeMode
	origID := guard.DebugAdminID
	guard.SafeMode = false
	guard.DebugAdminID = "debug1"
	t.Cleanup(func() {
		guard.SafeMode = origSafe
		guard.DebugAdminID = origID
	})

	// nil adminIDs = no Discord role confirmation
	err = syncRoles(database, nil, nil)
	require.NoError(t, err)

	var p models.Player
	err = database.NewSelect().Model(&p).Where("id = ?", "debug1").Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, models.ServerRolePlayer, p.Role, "debug admin must NOT be elevated without the Discord admin role when safe mode is off")
}

/*
When:

	SafeMode = OFF;
	DebugAdminID = set;
	User HAS the Discord admin role.

Expected:

	Elevated. Two-factor confirmed (env var + Discord role).
*/
func TestSyncRoles_DebugAdminElevatedWithDiscordRole(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	origSafe := guard.SafeMode
	origID := guard.DebugAdminID
	guard.SafeMode = false
	guard.DebugAdminID = "debug1"
	t.Cleanup(func() {
		guard.SafeMode = origSafe
		guard.DebugAdminID = origID
	})

	// "debug1" is in the Discord admin set — two-factor confirmed
	err = syncRoles(database, []string{"debug1"}, nil)
	require.NoError(t, err)

	var p models.Player
	err = database.NewSelect().Model(&p).Where("id = ?", "debug1").Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, models.ServerRoleAdmin, p.Role, "debug admin should be elevated when confirmed by Discord role")
}

/*
When:

	No player row for "ghost"

Expected:

	Sync should not panic or create one.
*/
func TestSyncRoles_DebugAdminUnregisteredIsNoOp(t *testing.T) {
	database := testutil.NewTestDB(t)

	withDebugAdmin(t, "ghost", func() {
		err := syncRoles(database, nil, nil)
		require.NoError(t, err)
	})
}

/*
Tests:

	Authorization.
	Prove security and sovereignty, even whilst elevated as debug admin.

When:

	Player is admin but globally banned.
*/
func TestDebugAdmin_BannedAdminDenied(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRoleAdmin, true)).Exec(ctx)
	require.NoError(t, err)

	withDebugAdmin(t, "debug1", func() {
		for _, scope := range []Scope{ScopePlayer, ScopeMod, ScopeAdmin} {
			ok, err := Authorize(database, "debug1", scope, "")
			require.NoError(t, err)
			assert.False(t, ok, "banned debug admin must be denied for %s", scope)
		}
	})
}

/*
Deny Unregistered

When:

	DEBUG_ADMIN_ID is set, but the user has no player row in the database.

Expected:

	Authorization must be denied. Debug elevation does not create players.
*/
func TestDebugAdmin_UnregisteredDenied(t *testing.T) {
	database := testutil.NewTestDB(t)

	// DEBUG_ADMIN_ID is set, but the user has no player row.
	withDebugAdmin(t, "no-such-user", func() {
		ok, err := Authorize(database, "no-such-user", ScopeAdmin, "")
		require.NoError(t, err)
		assert.False(t, ok, "unregistered debug admin must be denied")
	})
}

/*
Enforce sovereignty: cannot claim other DM's campaigns.

When:

	Debug admin is elevated but is not the DM of the target campaign.

Expected:

	ScopeDM must be denied. Admin role never implies DM ownership.
*/
func TestDebugAdmin_CannotClaimDMOfOthersCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newPlayer("real-dm", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "real-dm")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("real-dm", "camp1", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	withDebugAdmin(t, "debug1", func() {
		ok, err := Authorize(database, "debug1", ScopeDM, "camp1")
		require.NoError(t, err)
		assert.False(t, ok, "debug admin must not pass ScopeDM for someone else's campaign")
	})
}

/*
Member not in campaign they did not join.

When:

	Debug admin is elevated but has no CampaignPlayer row for the target campaign.

Expected:

	ScopeMember must be denied. Admin role does not grant campaign membership.
*/

func TestDebugAdmin_NotMemberOfCampaignTheyDidNotJoin(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm1")).Exec(ctx)
	require.NoError(t, err)

	withDebugAdmin(t, "debug1", func() {
		ok, err := Authorize(database, "debug1", ScopeMember, "camp1")
		require.NoError(t, err)
		assert.False(t, ok, "debug admin must not pass ScopeMember for a campaign they didn't join")
	})
}

/*
Authorization: passes Admin and Mod scopes.

When:

	Debug admin is elevated, registered, and not banned.

Expected:

	ScopeAdmin and ScopeMod both pass (admin implies mod).
*/
func TestDebugAdmin_PassesAdminAndModScopes(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	// After sync elevation, the player's DB role is admin.
	_, err := database.NewInsert().Model(newPlayer("debug1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)

	withDebugAdmin(t, "debug1", func() {
		okAdmin, err := Authorize(database, "debug1", ScopeAdmin, "")
		require.NoError(t, err)
		assert.True(t, okAdmin, "debug admin should pass ScopeAdmin")

		okMod, err := Authorize(database, "debug1", ScopeMod, "")
		require.NoError(t, err)
		assert.True(t, okMod, "debug admin should pass ScopeMod (admin implies mod)")
	})
}
