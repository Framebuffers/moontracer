package interactions

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/testutil"
)

/*
Unit Testing
Campaign invite flow (select / accept / decline)

Covers the three handler types that implement invitations:

  - manageCampaignInviteSelect:	 DM picks a user from the dropdown.
  - campaignInviteAccept:        target clicks Accept on the invitation DM.
  - campaignInviteDecline:       target clicks Decline on the invitation DM.

Auth is ScopeDM for the select step, and pending-membership presence for
accept/decline. Slot overflow is enforced at select time.
*/

const testTargetID = "target-1"

/*
inviteFixture seeds a DM + a pending campaign with one DM membership row.

Parameters let each test tune what's missing:

  - a registered target,
  - a full-capacity campaign, or
  - an existing pending invitation.

slots=0: means "unlimited" (no overflow check fires).
*/
func inviteFixture(t *testing.T, database *bun.DB, slots int, registerTarget bool) *dispatch.Dispatcher {
	t.Helper()
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testDMID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)

	if registerTarget {
		_, err = database.NewInsert().Model(&models.Player{ID: testTargetID, Role: models.ServerRolePlayer}).Exec(ctx)
		require.NoError(t, err)
	}

	_, err = database.NewInsert().Model(&models.Campaign{
		ID:            testCampID,
		Name:          "Invite Test",
		Tag:           "invite-test",
		DungeonMaster: testDMID,
		IsApproved:    true,
		Slots:         slots,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testDMID,
		CampaignID: testCampID,
		Role:       models.RoleDM,
		Status:     models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	return dispatch.NewDispatcher(nil, 1)
}

/*
buildInviteSelect constructs a user-select InteractionCreate for the invite dropdown.
The select handler reads CustomID + Values.
GuildID is needed for the accept/decline buttons it builds.
*/
func buildInviteSelect(userID, guildID, campaignID string, selected []string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionMessageComponent,
			GuildID: guildID,
			User:    &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: messages.ManageInviteSelectPrefix + ":" + campaignID,
				Values:   selected,
			},
		},
	}
}

/*
buildInviteButton constructs a button-click InteractionCreate
with the accept/decline CustomID format (prefix:guildID:campaignID).
*/
func buildInviteButton(prefix, userID, guildID, campaignID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			User: &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      prefix + ":" + guildID + ":" + campaignID,
				ComponentType: discordgo.ButtonComponent,
			},
		},
	}
}

/*
DM invites a registered user.

When:

	The DM selects a registered, non-member target via the user-select dropdown.

Expected:

	A CampaignPlayer row with StatusPending is inserted; an invitation DM
	with Accept/Decline buttons is queued for the target.
*/
func TestInviteSelect_DMAuthorized(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 4, true)
	stub := testutil.NewStubSession(t)

	handler := &manageCampaignInviteSelect{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildInviteSelect(testDMID, testGuildID, testCampID, []string{testTargetID}))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)

	var target *models.CampaignPlayer
	for i := range players {
		if players[i].PlayerID == testTargetID {
			target = &players[i]
		}
	}
	require.NotNil(t, target, "pending membership row should be created for the target")
	assert.Equal(t, models.StatusPending, target.Status)
	assert.Equal(t, models.RolePlayer, target.Role)

	pending := dispatcher.Pending()
	require.Len(t, pending, 1, "one invitation DM should be queued")
	assert.Equal(t, testTargetID, pending[0].Target)
	require.NotEmpty(t, pending[0].Components, "invitation DM should carry Accept/Decline buttons")
}

/*
Non-DM cannot invite.

When:

	A plain player (not the campaign's DM) triggers the select handler via a
	forged CustomID.

Expected:

	No pending row is created; no DM is queued.
*/
func TestInviteSelect_NonDMRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 4, true)

	// The attacker is registered but is NOT the DM of testCampID.
	_, err := database.NewInsert().Model(&models.Player{ID: "attacker", Role: models.ServerRolePlayer}).Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &manageCampaignInviteSelect{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildInviteSelect("attacker", testGuildID, testCampID, []string{testTargetID}))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		assert.NotEqual(t, testTargetID, p.PlayerID, "no row should be created for target")
	}
	assert.Empty(t, dispatcher.Pending(), "no DM should be queued")
}

/*
Unregistered target is rejected pre-insert.

When:

	DM selects a user ID that has no row in the players table.

Expected:

	No CampaignPlayer row is created; no DM queued.
	The guard clause at campaign_invite.go:143 catches this before mutation.
*/
func TestInviteSelect_UnregisteredTargetGuarded(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 4, false) // note: target NOT registered
	stub := testutil.NewStubSession(t)

	handler := &manageCampaignInviteSelect{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildInviteSelect(testDMID, testGuildID, testCampID, []string{testTargetID}))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		assert.NotEqual(t, testTargetID, p.PlayerID, "no row should be created for an unregistered user")
	}
	assert.Empty(t, dispatcher.Pending())
}

/*
Full campaign rejects new invite.

When:

	Campaign has Slots=1, the DM already occupies that slot (active), and the
	DM tries to invite another player. CanOverflow is false.

Expected:

	No pending row created; no DM queued. Overflow guard fires before insert.
*/
func TestInviteSelect_SlotOverflow(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 1, true) // Slots=1, DM already active
	stub := testutil.NewStubSession(t)

	handler := &manageCampaignInviteSelect{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildInviteSelect(testDMID, testGuildID, testCampID, []string{testTargetID}))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		assert.NotEqual(t, testTargetID, p.PlayerID, "overflow guard should block new membership row")
	}
	assert.Empty(t, dispatcher.Pending())
}

/*
Accept promotes pending to active.

When:

	A CampaignPlayer with StatusPending exists for the target, and the target
	clicks Accept on the invitation DM.

Expected:

	The row's Status flips to StatusActive. No role assignment is attempted
	because the fixture leaves Campaign.RoleID empty.
*/
func TestInviteAccept_CreatesMembership(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 0, true)

	// Step: Seed the pending invitation that Accept is responding to.
	_, err := database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testTargetID,
		CampaignID: testCampID,
		Role:       models.RolePlayer,
		Status:     models.StatusPending,
	}).Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &campaignInviteAccept{db: database}
	handler.HandleComponents(stub.Session, buildInviteButton(messages.InviteAcceptPrefix, testTargetID, testGuildID, testCampID))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	var target *models.CampaignPlayer
	for i := range players {
		if players[i].PlayerID == testTargetID {
			target = &players[i]
		}
	}
	require.NotNil(t, target, "target membership row should still exist after accept")
	assert.Equal(t, models.StatusActive, target.Status, "pending should be promoted to active")
	_ = dispatcher // Note: fixture returns it but accept flow doesn't dispatch
}

/*
Decline removes the pending row.

When:

	A pending CampaignPlayer row exists for the target, and the target
	clicks Decline on the invitation DM.

Expected:

	The pending row is removed. No other rows are touched (DM's active row survives).
*/
func TestInviteDecline_RemovesPending(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := inviteFixture(t, database, 0, true)

	_, err := database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testTargetID,
		CampaignID: testCampID,
		Role:       models.RolePlayer,
		Status:     models.StatusPending,
	}).Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &campaignInviteDecline{db: database}
	handler.HandleComponents(stub.Session, buildInviteButton(messages.InviteDeclinePrefix, testTargetID, testGuildID, testCampID))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		assert.NotEqual(t, testTargetID, p.PlayerID, "pending row should be removed after decline")
	}

	// Note: DM's own active row must survive the decline.
	var dmPresent bool
	for _, p := range players {
		if p.PlayerID == testDMID && p.Status == models.StatusActive {
			dmPresent = true
		}
	}
	assert.True(t, dmPresent, "DM's active membership must not be touched by a target's decline")
	_ = dispatcher
}
