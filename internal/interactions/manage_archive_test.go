package interactions

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/testutil"
)

/*
Unit Testing
Manage -> Archive confirm flow

Covers manageArchiveConfirm:
	The destructive button at the end of the manage-menu archive sub-flow
	that calls commands.ArchiveCampaign and flips IsArchived.

Auth is ScopeDM. Already-archived campaigns short-circuit at CanMutate()
so the second click is a no-op.
*/

/*
archiveFixture seeds a registered DM and a campaign owned by them, with a DM
membership row.

The alreadyArchived flag pre-flips IsArchived so tests can exercise the no-op branch.
*/
func archiveFixture(t *testing.T, database *bun.DB, alreadyArchived bool) {
	t.Helper()
	ctx := context.Background()

	_, err := database.NewInsert().Model(&models.Player{ID: testDMID, Role: models.ServerRolePlayer}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.Campaign{
		ID:            testCampID,
		Name:          "Archive Test",
		Tag:           "archive-test",
		DungeonMaster: testDMID,
		IsApproved:    true,
		IsArchived:    alreadyArchived,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   testDMID,
		CampaignID: testCampID,
		Role:       models.RoleDM,
		Status:     models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)
}

/*
buildArchiveConfirm constructs a button-click InteractionCreate matching
the manage_archive_confirm CustomID format (prefix:campaignID).
*/
func buildArchiveConfirm(userID, campaignID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionMessageComponent,
			GuildID: testGuildID,
			User:    &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      messages.ManageArchiveConfirmID + ":" + campaignID,
				ComponentType: discordgo.ButtonComponent,
			},
		},
	}
}

/*
DM archives their own campaign.

When:

	The campaign's DM clicks the Archive Confirm button.

Expected:

	Campaign.IsArchived flips to true; the DM's active membership is moved to
	StatusFinished by the ArchiveCampaign cascade.
*/
func TestManageArchiveConfirm_DMAuthorized(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	archiveFixture(t, database, false)
	stub := testutil.NewStubSession(t)

	handler := &manageArchiveConfirm{db: database}
	handler.HandleComponents(stub.Session, buildArchiveConfirm(testDMID, testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.True(t, reloaded.IsArchived, "campaign should be archived after DM confirm")

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		if p.PlayerID == testDMID {
			assert.Equal(t, models.StatusFinished, p.Status, "DM membership should be finished after archive cascade")
		}
	}
}

/*
Plain player cannot archive someone else's campaign.

When:

	A non-DM player triggers the confirm via a forged CustomID.

Expected:

	Campaign stays unarchived; DM's membership row is untouched.
*/
func TestManageArchiveConfirm_NonDMRejected(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	archiveFixture(t, database, false)

	_, err := database.NewInsert().Model(&models.Player{ID: "attacker", Role: models.ServerRolePlayer}).Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &manageArchiveConfirm{db: database}
	handler.HandleComponents(stub.Session, buildArchiveConfirm("attacker", testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.False(t, reloaded.IsArchived, "non-DM must not archive the campaign")

	players, err := models.GetCampaignPlayers(database, testCampID)
	require.NoError(t, err)
	for _, p := range players {
		if p.PlayerID == testDMID {
			assert.Equal(t, models.StatusActive, p.Status, "DM membership should remain active when archive is rejected")
		}
	}
}

/*
Already-archived campaign is a no-op on second click.

When:

	The DM clicks Archive Confirm on a campaign that's already archived
	(e.g. via a stale ephemeral menu).

Expected:

	No second-pass mutation. The CanMutate() guard short-circuits before
	ArchiveCampaign runs; ArchivedReason from the prior run is preserved.
*/
func TestManageArchiveConfirm_AlreadyArchivedNoOp(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	database := testutil.NewTestDB(t)
	archiveFixture(t, database, true) // Note: pre-archived

	// Step: pin a sentinel reason so we can detect any second-pass overwrite.
	_, err := database.NewUpdate().Model((*models.Campaign)(nil)).
		Set("archived_reason = ?", "first archive").
		Where("id = ?", testCampID).
		Exec(context.Background())
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &manageArchiveConfirm{db: database}
	handler.HandleComponents(stub.Session, buildArchiveConfirm(testDMID, testCampID))

	reloaded, err := db.GetByID[models.Campaign](database, testCampID)
	require.NoError(t, err)
	assert.True(t, reloaded.IsArchived, "still archived")
	assert.Equal(t, "first archive", reloaded.ArchivedReason, "reason must not be overwritten by a second confirm")
}
