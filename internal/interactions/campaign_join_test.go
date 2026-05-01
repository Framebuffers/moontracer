package interactions

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/testutil"
)

/*
Unit Testing
Westmarch FCFS join flow with session-capacity tripwire.

The tripwire is a soft alert: when the (N+1)th player joins past
SessionCapacity, the join still succeeds, but a DM is dispatched to the
campaign DM and the joining player gets an over-capacity ephemeral notice.

The tripwire is westmarch-only. Non-westmarch campaigns retain the existing
hard-rejection behaviour at Slots cap.
*/

const (
	testJoinerID = "joiner-1"
)

/*
westmarchJoinFixture seeds:

  - the DM (testDMID),
  - the joiner (testJoinerID),
  - a westmarch campaign (testCampID, IsApproved=true, IsOpen=true) with
    Slots=MaxInt32 and the supplied SessionCapacity,
  - the DM as an active CampaignPlayer,
  - prefillPlayers extra dummy active members so we can position the joiner
    relative to capacity.

prefillPlayers does NOT include the DM (DM is always seeded). To test the
3rd joiner against capacity=2 with the DM filling seat 1, pass prefill=1
(which adds one extra active member, so DM + 1 prefill = 2 active, joiner
is the 3rd → tripwire).
*/
func westmarchJoinFixture(t *testing.T, database *bun.DB, sessionCapacity, prefillPlayers int) *dispatch.Dispatcher {
	t.Helper()
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testDMID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Player{ID: testJoinerID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Campaign{
		ID:              testCampID,
		Name:            "Westmarch Test",
		Tag:             "westmarch-test",
		DungeonMaster:   testDMID,
		IsApproved:      true,
		IsOpen:          true,
		IsWestmarch:     true,
		Slots:           math.MaxInt32,
		SessionCapacity: sessionCapacity,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testDMID,
		CampaignID: testCampID,
		Role:       models.RoleDM,
		Status:     models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	for n := 0; n < prefillPlayers; n++ {
		prefillID := "prefill-" + string(rune('a'+n))
		_, err = database.NewInsert().Model(&models.Player{ID: prefillID, Role: models.ServerRolePlayer}).Exec(ctx)
		require.NoError(t, err)
		_, err = database.NewInsert().Model(&models.CampaignPlayer{
			PlayerID:   prefillID,
			CampaignID: testCampID,
			Role:       models.RolePlayer,
			Status:     models.StatusActive,
		}).Exec(ctx)
		require.NoError(t, err)
	}

	return dispatch.NewDispatcher(nil, 1)
}

/*
buildJoinInteraction constructs a button-click InteractionCreate for the
"Join Campaign" button.

The handler reads i.Member.User.ID (not i.User).
*/
func buildJoinInteraction(userID, guildID, tag string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionMessageComponent,
			GuildID: guildID,
			Member: &discordgo.Member{
				User:  &discordgo.User{ID: userID},
				Roles: []string{},
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      "campaign_join:" + tag,
				ComponentType: discordgo.ButtonComponent,
			},
		},
	}
}

/*
Joining within session capacity is silent.

When:

	A westmarch with SessionCapacity=6 has 1 active member (the DM); a
	registered, non-banned player clicks Join.

Expected:

	The joiner is admitted (active row created), and no DM is dispatched
	to the campaign DM. The active count after admission (2) is still
	at-or-below capacity, so the tripwire never fires.
*/
func TestJoin_WestmarchUnderCapacityIsSilent(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := westmarchJoinFixture(t, database, 6, 0)
	stub := testutil.NewStubSession(t)

	handler := &campaignJoin{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildJoinInteraction(testJoinerID, testGuildID, "westmarch-test"))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	var joiner *models.CampaignPlayer
	for i := range players {
		if players[i].PlayerID == testJoinerID {
			joiner = &players[i]
		}
	}
	require.NotNil(t, joiner, "joiner should be admitted to the westmarch")
	assert.Equal(t, models.StatusActive, joiner.Status)

	assert.Empty(t, dispatcher.Pending(), "tripwire must not fire while under session capacity")
}

/*
Crossing session capacity admits the player and DMs the DM.

When:

	A westmarch with SessionCapacity=2 has 2 active members (DM + 1
	prefilled player). A registered, non-banned 3rd player clicks Join.

Expected:

	The 3rd player is admitted (active row created, engine never blocks).
	One DM is dispatched, targeted at the campaign DM, and its content
	includes the joiner mention, the campaign name, the new active count
	(3), and the configured capacity (2).
*/
func TestJoin_WestmarchOverCapacityFiresTripwire(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := westmarchJoinFixture(t, database, 2, 1) // DM + 1 prefill = 2 active
	stub := testutil.NewStubSession(t)

	handler := &campaignJoin{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildJoinInteraction(testJoinerID, testGuildID, "westmarch-test"))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	var joiner *models.CampaignPlayer
	for i := range players {
		if players[i].PlayerID == testJoinerID {
			joiner = &players[i]
		}
	}
	require.NotNil(t, joiner, "tripwire is admit-then-alert; the joiner must still be admitted")
	assert.Equal(t, models.StatusActive, joiner.Status)

	pending := dispatcher.Pending()
	require.Len(t, pending, 1, "exactly one over-capacity DM should be queued")
	assert.Equal(t, testDMID, pending[0].Target, "DM alert must be targeted at the campaign DM")
	assert.Contains(t, pending[0].Content, testJoinerID, "DM alert should mention the joining player")
	assert.Contains(t, pending[0].Content, "Westmarch Test", "DM alert should name the campaign")
	assert.True(t,
		strings.Contains(pending[0].Content, "3") && strings.Contains(pending[0].Content, "2"),
		"DM alert should include both the new active count (3) and the configured capacity (2)")
}

/*
Non-westmarch campaigns still hard-reject at Slots cap.

When:

	A non-westmarch campaign has Slots=1 and the DM occupies the only seat.
	A registered player clicks Join.

Expected:

	The joiner is NOT admitted (existing rejection path preserved). No DM
	is dispatched (tripwire is westmarch-only).
*/
func TestJoin_NonWestmarchHardRejectionPreserved(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testDMID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(&models.Player{ID: testJoinerID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(&models.Campaign{
		ID:            testCampID,
		Name:          "Regular Campaign",
		Tag:           "regular-test",
		DungeonMaster: testDMID,
		IsApproved:    true,
		IsOpen:        true,
		IsWestmarch:   false,
		Slots:         1,
	}).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testDMID,
		CampaignID: testCampID,
		Role:       models.RoleDM,
		Status:     models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	dispatcher := dispatch.NewDispatcher(nil, 1)
	stub := testutil.NewStubSession(t)

	handler := &campaignJoin{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildJoinInteraction(testJoinerID, testGuildID, "regular-test"))

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		assert.NotEqual(t, testJoinerID, p.PlayerID, "non-westmarch full campaign must reject the joiner")
	}
	assert.Empty(t, dispatcher.Pending(), "tripwire must not fire on non-westmarch campaigns")
}
