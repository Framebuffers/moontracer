package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moontracer/internal/manager/models"
	"moontracer/internal/testutil"
)

// --- TESTS ---
// --- Approval auth: who can approve/deny campaigns ---

func TestApprovalAuth_ModCanApprove(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("mod1", models.ServerRoleMod, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "mod1", ScopeMod, "")
	require.NoError(t, err)
	assert.True(t, ok, "mod should be able to approve/deny campaigns")
}

func TestApprovalAuth_AdminCanApprove(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("admin1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "admin1", ScopeMod, "")
	require.NoError(t, err)
	assert.True(t, ok, "admin should be able to approve/deny campaigns (admin implies mod)")
}

func TestApprovalAuth_PlayerCannotApprove(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "user1", ScopeMod, "")
	require.NoError(t, err)
	assert.False(t, ok, "regular player should not be able to approve/deny campaigns")
}

func TestApprovalAuth_BannedModCannotApprove(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("banned-mod", models.ServerRoleMod, true)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "banned-mod", ScopeMod, "")
	require.NoError(t, err)
	assert.False(t, ok, "globally banned mod should not be able to approve/deny campaigns")
}

func TestApprovalAuth_BannedAdminCannotApprove(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("banned-admin", models.ServerRoleAdmin, true)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "banned-admin", ScopeMod, "")
	require.NoError(t, err)
	assert.False(t, ok, "globally banned admin should not be able to approve/deny campaigns")
}

func TestApprovalAuth_UnregisteredUserCannotApprove(t *testing.T) {
	database := testutil.NewTestDB(t)

	ok, err := Authorize(database, "ghost", ScopeMod, "")
	require.NoError(t, err)
	assert.False(t, ok, "unregistered user should not be able to approve/deny campaigns")
}
