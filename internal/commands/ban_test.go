package commands

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/testutil"
)

/*
Unit Testing
/ban command — hierarchy and cascade

Covers banCommand.Execute:
	global-ban flag flip + audit entry insert.
	Campaign memberships are intentionally NOT touched (DM sovereignty model: the global
	ban flag blocks auth.go, so cascades are cosmetic elsewhere).

Auth is ScopeMod on the invoker, plus role-weight strict-greater-than on the
target.

Banned status and self-ban are rejected before persistence.
*/

const (
	testBanInvokerID = "mod-1"
	testBanTargetID  = "player-1"
)

/*
banFixture seeds invoker + target players with the given roles and ban state.
*/
func banFixture(t *testing.T, database *bun.DB, invokerRole, targetRole models.ServerRole, targetAlreadyBanned bool) {
	t.Helper()
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testBanInvokerID, Role: invokerRole}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Player{
		ID:             testBanTargetID,
		Role:           targetRole,
		PlayerIsBanned: targetAlreadyBanned,
	}).Exec(ctx)
	require.NoError(t, err)
}

/*
buildBanInteraction constructs an ApplicationCommand InteractionCreate with
the two /ban options (player + optional reason) and a Member-shaped invoker.
*/
func buildBanInteraction(invokerID, targetID, reason string) *discordgo.InteractionCreate {
	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{
			Name:  "player",
			Type:  discordgo.ApplicationCommandOptionUser,
			Value: targetID,
		},
	}
	if reason != "" {
		options = append(options, &discordgo.ApplicationCommandInteractionDataOption{
			Name:  "reason",
			Type:  discordgo.ApplicationCommandOptionString,
			Value: reason,
		})
	}

	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:   discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{User: &discordgo.User{ID: invokerID}},
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    messages.BanCommandName,
				Options: options,
			},
		},
	}
}

/*
Mod bans a plain player with a reason.

When:

	A mod invokes /ban on a non-mod target with a reason string.

Expected:

	Target.PlayerIsBanned flips to true; Target.PlayerBanReason is stored;
	an AuditBan row is written attributing the action to the mod.
*/
func TestBan_ModBansPlayer(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	banFixture(t, database, models.ServerRoleMod, models.ServerRolePlayer, false)
	stub := testutil.NewStubSession(t)

	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanTargetID, "spamming"))

	target, err := db.GetByID[models.Player](database, testBanTargetID)
	require.NoError(t, err)
	assert.True(t, target.PlayerIsBanned, "target should be globally banned")
	assert.Equal(t, "spamming", target.PlayerBanReason)

	var audits []models.AuditEntry
	err = database.NewSelect().Model(&audits).Where("player_id = ?", testBanTargetID).Scan(context.Background())
	require.NoError(t, err)
	require.Len(t, audits, 1, "exactly one audit entry should be written")
	assert.Equal(t, models.AuditBan, audits[0].Action)
	assert.Equal(t, testBanInvokerID, audits[0].AuthorID)
}

/*
Cannot ban yourself.

When:

	A mod passes their own ID as the target.

Expected:

	Self-ban guard fires before auth; no ban flag flip, no audit entry.
*/
func TestBan_SelfBanRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	banFixture(t, database, models.ServerRoleMod, models.ServerRolePlayer, false)
	stub := testutil.NewStubSession(t)

	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanInvokerID, ""))

	invoker, err := db.GetByID[models.Player](database, testBanInvokerID)
	require.NoError(t, err)
	assert.False(t, invoker.PlayerIsBanned, "self-ban must not flip the flag")
}

/*
Plain player cannot invoke /ban.

When:

	A player (non-mod) runs /ban.

Expected:

	ScopeMod auth rejects before role-weight check. Target is untouched.
*/
func TestBan_PlainPlayerRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	banFixture(t, database, models.ServerRolePlayer, models.ServerRolePlayer, false)
	stub := testutil.NewStubSession(t)

	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanTargetID, ""))

	target, err := db.GetByID[models.Player](database, testBanTargetID)
	require.NoError(t, err)
	assert.False(t, target.PlayerIsBanned, "plain player must not ban anyone")
}

/*
Mod cannot ban equal-rank mod.

When:

	A mod invokes /ban on another mod (same weight).

Expected:

	Weight strict-greater-than guard rejects; target is untouched. This is
	the rule that protects peers from each other.
*/
func TestBan_EqualRankRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	banFixture(t, database, models.ServerRoleMod, models.ServerRoleMod, false)
	stub := testutil.NewStubSession(t)

	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanTargetID, ""))

	target, err := db.GetByID[models.Player](database, testBanTargetID)
	require.NoError(t, err)
	assert.False(t, target.PlayerIsBanned, "mod must not ban an equal-rank mod")
}

/*
Already-banned target is a no-op.

When:

	A mod tries to ban someone whose PlayerIsBanned is already true.

Expected:

	Bail-out before persistence; no duplicate audit entry is written.
*/
func TestBan_AlreadyBannedBailOut(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	banFixture(t, database, models.ServerRoleMod, models.ServerRolePlayer, true)
	stub := testutil.NewStubSession(t)

	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanTargetID, "second try"))

	var audits []models.AuditEntry
	err := database.NewSelect().Model(&audits).Where("player_id = ?", testBanTargetID).Scan(context.Background())
	require.NoError(t, err)
	assert.Empty(t, audits, "no audit entry should be written for an already-banned target")
}

/*
Unregistered target rejected.

When:

	A mod runs /ban on a user ID with no Player row.

Expected:

	BanTargetNotFound guard fires after invoker auth; no row is created for
	the unregistered user.
*/
func TestBan_UnregisteredTargetRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	// Step: seed only the invoker; target has no Player row.
	_, err := database.NewInsert().Model(&models.Player{ID: testBanInvokerID, Role: models.ServerRoleMod}).Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &banCommand{db: database}
	handler.Execute(stub.Session, buildBanInteraction(testBanInvokerID, testBanTargetID, ""))

	_, err = db.GetByID[models.Player](database, testBanTargetID)
	assert.Error(t, err, "no Player row should be created for an unregistered target")
}
