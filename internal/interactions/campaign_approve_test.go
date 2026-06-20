package interactions

import (
	"context"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/testutil"
)

/*
Unit Testing
Campaign approval/denial handlers

These tests exercise the ComponentHandler path end-to-end, up to the Discord session boundary.
The session is swapped for testutil.StubSession so REST calls are captured in memory
rather than hitting Discord.

SafeMode is enabled for every test so guard-wrapped mutations (channel/thread creation during
approval) short-circuit.

Dispatcher inspection

	Construct the dispatcher without calling Start(), so Push()'d DMs accumulate in the
	pending stack where Pending() can read them.

	This keeps tests deterministic. No worker goroutines, therefore no session sends.
*/

const (
	testGuildID = "guild-1"
	testModID   = "mod-1"
	testDMID    = "dm-1"
	testCampID  = "camp-1"
)

/*
approveFixture seeds a minimal DB with a mod, a DM, and a pending campaign
owned by that DM.

Returns the dispatcher so tests can inspect queued DMs.
*/
func approveFixture(t *testing.T, database *bun.DB, modRole models.ServerRole, modBanned bool) *dispatch.Dispatcher {
	t.Helper()
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testModID, Role: modRole, PlayerIsBanned: modBanned}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Player{ID: testDMID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Campaign{
		ID:            testCampID,
		Name:          "Test Campaign",
		Tag:           "test",
		DungeonMaster: testDMID,
		IsApproved:    false,
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
buildApproveInteraction constructs a button-click InteractionCreate targeting
campaign_approve with the DM-interaction shape (User populated, Member nil).
*/
func buildApproveInteraction(userID, guildID, campaignID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			User: &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      messages.CampaignApprovePrefix + ":" + guildID + ":" + campaignID,
				ComponentType: discordgo.ButtonComponent,
			},
		},
	}
}

/*
buildDenyModalInteraction constructs a modal-submit InteractionCreate with
the deny-reason text field populated.
*/
func buildDenyModalInteraction(userID, guildID, campaignID, reason string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionModalSubmit,
			User: &discordgo.User{ID: userID},
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: messages.CampaignDenyModalPrefix + ":" + guildID + ":" + campaignID,
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						&discordgo.TextInput{
							CustomID: messages.CampaignDenyReasonFieldID,
							Value:    reason,
						},
					}},
				},
			},
		},
	}
}

/*
Mod approves a pending campaign.

When:

	A mod clicks the Approve button on a pending campaign's DM.

Expected:

	Campaign.IsApproved flips to true; a DM is queued to the campaign's DM
	announcing approval.
*/
func TestCampaignApprove_ModAuthorized(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := approveFixture(t, database, models.ServerRoleMod, false)
	stub := testutil.NewStubSession(t)

	handler := &campaignApprove{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildApproveInteraction(testModID, testGuildID, testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.True(t, reloaded.IsApproved, "campaign should be approved after mod click")

	pending := dispatcher.Pending()
	require.Len(t, pending, 1, "one approval DM should be queued")
	assert.Equal(t, testDMID, pending[0].Target, "DM should be addressed to the campaign's DM")
	assert.Contains(t, pending[0].Content, "Test Campaign")
}

/*
Plain player cannot approve.

When:

	A regular player (ServerRolePlayer) clicks Approve- e.g. via a forged
	CustomID or leaked DM button.

Expected:

	Campaign stays unapproved; no DM queued.
*/
func TestCampaignApprove_UnauthorizedRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := approveFixture(t, database, models.ServerRolePlayer, false)
	stub := testutil.NewStubSession(t)

	handler := &campaignApprove{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildApproveInteraction(testModID, testGuildID, testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.False(t, reloaded.IsApproved, "plain player must not approve")
	assert.Empty(t, dispatcher.Pending(), "no DM should be queued on rejection")
}

/*
Banned mod cannot approve.

When:

	A user with ServerRoleMod but PlayerIsBanned=true clicks Approve.

Expected:

	Global ban overrides role; campaign stays unapproved.
*/
func TestCampaignApprove_BannedModRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := approveFixture(t, database, models.ServerRoleMod, true)
	stub := testutil.NewStubSession(t)

	handler := &campaignApprove{db: database, dispatcher: dispatcher}
	handler.HandleComponents(stub.Session, buildApproveInteraction(testModID, testGuildID, testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.False(t, reloaded.IsApproved, "globally-banned mod must not approve")
	assert.Empty(t, dispatcher.Pending())
}

/*
Deny modal captures the reason and forwards it to the DM.

When:

	A mod submits the deny modal with a typed reason.

Expected:

	Campaign row (and its CampaignPlayers) is deleted; a DM is queued to the
	campaign's DM containing the reason text.
*/
func TestCampaignDenyModal_ReasonCaptured(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := approveFixture(t, database, models.ServerRoleMod, false)
	stub := testutil.NewStubSession(t)

	reason := "duplicate of an existing campaign"
	handler := &campaignDenyModal{db: database, dispatcher: dispatcher}
	handler.HandleModal(stub.Session, buildDenyModalInteraction(testModID, testGuildID, testCampID, reason))

	_, err := db.GetByID[models.Campaign](database, testCampID)
	assert.Error(t, err, "campaign row should be deleted after denial")

	pending := dispatcher.Pending()
	require.Len(t, pending, 1, "one denial DM should be queued")
	assert.Equal(t, testDMID, pending[0].Target)
	assert.True(t, strings.Contains(pending[0].Content, reason), "denial DM should include the mod's reason, got %q", pending[0].Content)
}

/*
Non-mod cannot deny.

When:

	A plain player submits the deny modal via a forged CustomID.

Expected:

	Campaign is still present; no DM queued.
*/
func TestCampaignDenyModal_UnauthorizedRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	dispatcher := approveFixture(t, database, models.ServerRolePlayer, false)
	stub := testutil.NewStubSession(t)

	handler := &campaignDenyModal{db: database, dispatcher: dispatcher}
	handler.HandleModal(stub.Session, buildDenyModalInteraction(testModID, testGuildID, testCampID, "no reason given"))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err, "campaign must survive an unauthorized deny attempt")
	assert.False(t, reloaded.IsApproved)
	assert.Empty(t, dispatcher.Pending())
}
