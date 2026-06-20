package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/testutil"
)

/*
Helper functions
*/

func newPlayer(id string, role models.ServerRole, banned bool) *models.Player {
	return &models.Player{
		ID:             id,
		Role:           role,
		PlayerIsBanned: banned,
	}
}

func newCampaignPlayer(playerID, campaignID string, role models.Role, status models.CampaignPlayerStatus) *models.CampaignPlayer {
	return &models.CampaignPlayer{
		PlayerID:   playerID,
		CampaignID: campaignID,
		Role:       role,
		Status:     status,
	}
}

func newCampaign(id, name, tag, dmID string) *models.Campaign {
	return &models.Campaign{
		ID:            id,
		Name:          name,
		Tag:           tag,
		DungeonMaster: dmID,
	}
}

/*
Unit Testing: enforce authorization and sovereignty rules.
*/

/*
Cross-campaign isolation truth table:

	|     campA     |     campB     |
	+---------------+---------------+
	usr1|       T       |       F       |
	----+---------------+---------------+
	usr2|       F       |       T       |
	----+---------------+---------------+

When:

	Two DMs each own one campaign.

Expected:

	Each DM is authorized only for their own campaign, never the other's.
*/
func TestCrossCampaignIsolation(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("dm1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(newPlayer("dm2", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(newCampaign("campA", "Test Campaign 1", "test1", "dm1")).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(newCampaign("campB", "Test Campaign 2", "test2", "dm2")).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(newCampaignPlayer("dm1", "campA", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(newCampaignPlayer("dm2", "campB", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	authDM1campaignA, err := Authorize(database, "dm1", ScopeDM, "campA")
	require.NoError(t, err)
	assert.True(t, authDM1campaignA, "DM1 should be the master of CampaignA. DM1 is not authorized as master of this campaign.")

	authDM2campaignB, err := Authorize(database, "dm2", ScopeDM, "campB")
	require.NoError(t, err)
	assert.True(t, authDM2campaignB, "DM2 should be the master of CampaignB. DM2 is not authorized as master of this campaign.")

	authDM1campaignB, err := Authorize(database, "dm1", ScopeDM, "campB")
	require.NoError(t, err)
	assert.False(t, authDM1campaignB, "DM1 cannot be the master of CampaignB. DM1 is not authorized as a master of this campaign.")

	authDM2campaignA, err := Authorize(database, "dm2", ScopeDM, "campA")
	require.NoError(t, err)
	assert.False(t, authDM2campaignA, "DM2 cannot be the master of CampaignA. DM2 is not authorized as the master of this campaign.")
}

/*
Unregistered user.

When:

	User ID does not exist in the players table.

Expected:

	Authorization denied for any scope.
*/
func TestAuthorize_UnregisteredUser(t *testing.T) {
	database := testutil.NewTestDB(t)

	ok, err := Authorize(database, "unknown-user", ScopePlayer, "")
	require.NoError(t, err)
	assert.False(t, ok, "unregistered user should not be authorized")
}

/*
Registered player.

When:

	User exists in the players table, is not banned, role = player.

Expected:

	ScopePlayer passes.
*/
func TestAuthorize_RegisteredPlayer(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "user1", ScopePlayer, "")
	require.NoError(t, err)
	assert.True(t, ok, "registered player should be authorized for ScopePlayer")
}

/*
Globally banned player.

When:

	User has admin role but is globally banned.

Expected:

	All scopes denied. Global ban overrides every role.
*/
func TestAuthorize_GloballyBannedPlayer(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("banned1", models.ServerRoleAdmin, true)).Exec(ctx)
	require.NoError(t, err)

	scopes := []Scope{ScopePlayer, ScopeMod, ScopeAdmin}
	for _, scope := range scopes {
		ok, err := Authorize(database, "banned1", scope, "")
		require.NoError(t, err)
		assert.False(t, ok, "banned player should fail %s", scope)
	}
}

/*
Active campaign member.

When:

	Player has an active CampaignPlayer row for the target campaign.

Expected:

	ScopeMember passes.
*/
func TestAuthorize_ActiveCampaignMember(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Test Campaign", "test", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("user1", "camp1", models.RolePlayer, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "user1", ScopeMember, "camp1")
	require.NoError(t, err)
	assert.True(t, ok, "active member should be authorized")
}

/*
Inactive campaign member.

When:

	Player has a CampaignPlayer row but with StatusHiatus.

Expected:

	ScopeMember denied. Only active members pass.
*/
func TestAuthorize_InactiveCampaignMember(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Test Campaign", "test", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("user1", "camp1", models.RolePlayer, models.StatusHiatus)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "user1", ScopeMember, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "hiatus member should not be authorized as active member")
}

/*
DM of campaign.

When:

	Player has a CampaignPlayer row with RoleDM for the target campaign.

Expected:

	ScopeDM passes.
*/
func TestAuthorize_DMOfCampaign(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("dm1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "DM Campaign", "dmcamp", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("dm1", "camp1", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "dm1", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.True(t, ok, "DM of campaign should be authorized")
}

/*
Player is not DM.

When:

	Player has a CampaignPlayer row with RolePlayer (not RoleDM).

Expected:

	ScopeDM denied. Campaign membership does not imply DM ownership.
*/
func TestAuthorize_PlayerIsNotDM(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("user1", "camp1", models.RolePlayer, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "user1", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "player role should not authorize as DM")
}

/*
Mod role.

When:

	Player has ServerRoleMod.

Expected:

	ScopeMod passes.
*/
func TestAuthorize_ModRole(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("mod1", models.ServerRoleMod, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "mod1", ScopeMod, "")
	require.NoError(t, err)
	assert.True(t, ok, "mod should be authorized for ScopeMod")
}

/*
Admin implies mod.

When:

	Player has ServerRoleAdmin.

Expected:

	ScopeMod passes. Admin is a superset of mod.
*/
func TestAuthorize_AdminImpliesMod(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("admin1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "admin1", ScopeMod, "")
	require.NoError(t, err)
	assert.True(t, ok, "admin should pass ScopeMod check (admin implies mod)")
}

/*
Admin scope.

When:

	Player has ServerRoleAdmin.

Expected:

	ScopeAdmin passes.
*/
func TestAuthorize_AdminScope(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("admin1", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "admin1", ScopeAdmin, "")
	require.NoError(t, err)
	assert.True(t, ok, "admin should be authorized for ScopeAdmin")
}

/*
Mod is not admin.

When:

	Player has ServerRoleMod.

Expected:

	ScopeAdmin denied. Mod does not imply admin.
*/
func TestAuthorize_ModIsNotAdmin(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("mod1", models.ServerRoleMod, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "mod1", ScopeAdmin, "")
	require.NoError(t, err)
	assert.False(t, ok, "mod should not be authorized for ScopeAdmin")
}

/*
AuthorizeAny: DM or Mod- user is DM.

When:

	Player is the DM of the target campaign, with ServerRolePlayer.

Expected:

	AuthorizeAny(DM, Mod) passes via the DM path.
*/
func TestAuthorizeAny_DMOrMod_IsDM(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("dm1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("dm1", "camp1", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := AuthorizeAny(database, "dm1", "camp1", ScopeDM, ScopeMod)
	require.NoError(t, err)
	assert.True(t, ok, "DM should pass AuthorizeAny(DM, Mod)")
}

/*
Sovereignty: mod cannot claim ScopeDM.

When:

	A mod is a regular player in a campaign they do not DM.
	A second mod is not in the campaign at all.

Expected:

	ScopeDM denied for both. Server role never overrides campaign ownership.
*/
func TestAuthorize_ModCannotScopeDM(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("dm", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newPlayer("mod", models.ServerRoleMod, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("dm", "camp1", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	// Mod as a regular player in the campaign- cannot claim DM.
	_, err = database.NewInsert().Model(newCampaignPlayer("mod", "camp1", models.RolePlayer, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := Authorize(database, "mod", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "Mod who is a campaign member (RolePlayer) must not pass ScopeDM")

	// Mod not in the campaign at all- still cannot claim DM.
	_, err = database.NewInsert().Model(newPlayer("mod2", models.ServerRoleMod, false)).Exec(ctx)
	require.NoError(t, err)

	ok, err = Authorize(database, "mod2", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "Mod who is not in the campaign must not pass ScopeDM")
}

/*
Sovereignty: admin cannot claim ScopeDM.

When:

	An admin is not in the campaign.
	Then the same admin is added as a regular player.

Expected:

	ScopeDM denied in both cases. Admin privilege never implies DM ownership.
*/
func TestAuthorize_AdminCannotScopeDM(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("dm", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newPlayer("admin", models.ServerRoleAdmin, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("dm", "camp1", models.RoleDM, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	// Admin not in the campaign- cannot claim DM.
	ok, err := Authorize(database, "admin", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "Admin who is not in the campaign must not pass ScopeDM")

	// Admin as a regular player in the campaign- still cannot claim DM.
	_, err = database.NewInsert().Model(newCampaignPlayer("admin", "camp1", models.RolePlayer, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err = Authorize(database, "admin", ScopeDM, "camp1")
	require.NoError(t, err)
	assert.False(t, ok, "Admin who is a campaign member (RolePlayer) must not pass ScopeDM")
}

/*
AuthorizeAny: neither scope matches.

When:

	A plain player is a member of a campaign but is neither DM nor mod.

Expected:

	AuthorizeAny(DM, Mod) denied.
*/
func TestAuthorizeAny_Neither(t *testing.T) {
	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(newPlayer("user1", models.ServerRolePlayer, false)).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaign("camp1", "Campaign", "camp", "dm1")).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(newCampaignPlayer("user1", "camp1", models.RolePlayer, models.StatusActive)).Exec(ctx)
	require.NoError(t, err)

	ok, err := AuthorizeAny(database, "user1", "camp1", ScopeDM, ScopeMod)
	require.NoError(t, err)
	assert.False(t, ok, "plain player should fail AuthorizeAny(DM, Mod)")
}
