package interactions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/testutil"
)

/*
Coverage for the post-upload Apply / Discard / Postcreate-assign handlers.

Critical issue: the Apply / Discard CustomIDs encode (guildID, playerID, token).
Both handlers extract playerID from the CustomID, NOT from the click's
helpers.GetUserID(i). That means anyone who clicks the button (or fabricates
the CustomID via interaction replay) acts AS that playerID:

	- tokenApplyModal inserts a Media row owned by the CustomID's playerID.
	- tokenDiscardHandler deletes the temp files belonging to the CustomID's
	  playerID.

The token UUID is unguessable, so practical exploitation requires the
attacker to see the original Apply button (e.g. via a leaked ephemeral
screenshot or a bug surfacing the CustomID). The fix is to compare
helpers.GetUserID(i) against parts[2] in both handlers.

The postcreate select handler has a different shape: the UPDATE WHERE filters
by the clicker's userID, so an attacker can only stamp a victim's mediaID
into THEIR OWN CampaignPlayer row: cosmetic, but still a cross-owner data
reference that the gallery flow's fix should also close here.
*/

const (
	tokConfirmGuildID  = "guild-1"
	tokConfirmVictimID = "victim-1"
	tokConfirmAtkID    = "attacker-2"
	tokConfirmTokenID  = "uuid-token-1"
)

func writePreviewFiles(t *testing.T, dataDir, guildID, playerID, token string) []string {
	t.Helper()
	dir := filepath.Join(dataDir, guildID, "tokens", playerID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	paths := []string{
		filepath.Join(dir, "src_"+token+".png"),
		filepath.Join(dir, "out_"+token+".png"),
	}
	for _, p := range paths {
		require.NoError(t, os.WriteFile(p, []byte("preview"), 0o644))
	}
	return paths
}

/*
Attacker forges a token_apply_modal submission targeting the victim's preview.

When:

	The victim ran /uploadtoken (preview written to disk). The attacker
	submits a modal with CustomID embedding the victim's playerID + token.

Expected:

	No Media row is created with OwnerID=victim.

Development Note (v0.12.6, 20260511):

	Currently FAILS:
		modal pulls playerID from the CustomID, so the attacker writes a Media row
		on the victim's behalf pointing at the on-disk preview.
*/
func TestTokenApplyModal_RejectsCrossUser(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	writePreviewFiles(t, dataDir, tokConfirmGuildID, tokConfirmVictimID, tokConfirmTokenID)

	stub := testutil.NewStubSession(t)
	handler := &tokenApplyModal{db: database, dataDir: dataDir, mediaBaseURL: "http://cdn"}

	customID := fmt.Sprintf("%s:%s:%s:%s",
		messages.TokenApplyModalPrefix, tokConfirmGuildID, tokConfirmVictimID, tokConfirmTokenID)
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionModalSubmit,
			User: &discordgo.User{ID: tokConfirmAtkID},
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: customID,
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						&discordgo.TextInput{CustomID: messages.TokenNameFieldID, Value: "stolen"},
					}},
				},
			},
		},
	}
	handler.HandleModal(stub.Session, interaction)

	count, err := database.NewSelect().Model((*models.Media)(nil)).
		Where("owner_id = ?", tokConfirmVictimID).Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, count,
		"attacker created a Media row owned by the victim — TokenApplyModal must check helpers.GetUserID == playerID")
}

/*
Attacker forges a token_discard targeting the victim's preview.

When:

	The victim has a preview on disk awaiting Apply. The attacker submits
	a discard CustomID embedding the victim's playerID + token.

Expected:

	The victim's preview files remain on disk.

Development Note (v0.12.6, 20260511):

	Currently FAILS:
		discard playerID from the CustomID and unlinks anything matching the
		pattern.
*/
func TestTokenDiscard_RejectsCrossUser(t *testing.T) {
	dataDir := t.TempDir()
	paths := writePreviewFiles(t, dataDir, tokConfirmGuildID, tokConfirmVictimID, tokConfirmTokenID)

	stub := testutil.NewStubSession(t)
	handler := &tokenDiscardHandler{dataDir: dataDir, mediaBaseURL: "http://cdn"}

	customID := fmt.Sprintf("%s:%s:%s:%s",
		messages.TokenDiscardPrefix, tokConfirmGuildID, tokConfirmVictimID, tokConfirmTokenID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokConfirmAtkID, customID))

	for _, p := range paths {
		_, err := os.Stat(p)
		assert.NoError(t, err, "victim's preview file %s deleted by attacker", p)
	}
}

/*
Attacker uses the postcreate select to stamp the victim's mediaID into their
own roster row.

When:

	The attacker is a player in some campaign and submits a postcreate-select
	whose mediaID points at the victim's token.

Expected:

	No CampaignPlayer row anywhere references the victim's mediaID.

Development Note (v0.12.6, 20260511):

	Currently FAILS:
		handler updates by (clicker, campaign) without verifying the
		media belongs to the clicker.
*/
func TestPlayerTokenPostcreate_RejectsCrossOwnerMedia(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	ctx := context.Background()
	const campaignID = "camp-x"
	_, err := database.NewInsert().Model(&models.Campaign{
		ID: campaignID, Name: "X", Tag: "x",
		DungeonMaster: tokConfirmAtkID, IsApproved: true,
	}).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID: tokConfirmAtkID, CampaignID: campaignID,
		Role: models.RolePlayer, Status: models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &playerTokenPostcreateSelectHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenPostcreateSelectPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session,
		buildSelectInteraction(tokConfirmAtkID, customID, []string{campaignID}))

	count, err := database.NewSelect().Model((*models.CampaignPlayer)(nil)).
		Where("media_id = ?", tokVictimMID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count,
		"attacker stamped victim's mediaID into their own roster row — postcreate must verify media.OwnerID == userID")
}
