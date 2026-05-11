package interactions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/testutil"
)

/*
Ownership-guard coverage for the player token gallery.

Every handler in player_tokens.go loads the target Media row by ID alone:

	media, err := db.GetByID[models.Media](h.db, mediaID)

There is no `WHERE owner_id = ?` clause and no post-load comparison against
helpers.GetUserID(i). A user who learns or guesses another user's mediaID
(via leaked link, message scrape, or simply iterating UUIDs in dev) can:

	- Assign someone else's token to their own campaign roster
	- Download the raw PNG bytes
	- Delete the record + file

These tests assert the *correct* behavior: an interaction from a non-owner
must not mutate state, must not return file bytes, and should respond with
an error message. They will FAIL on current main — that's the point. They
document the missing guard for the fix PR.
*/

const (
	tokOwnerID    = "owner-1"
	tokAttackerID = "attacker-2"
	tokVictimMID  = "media-victim"
)

func seedVictimToken(t *testing.T, database *bun.DB, dataDir string) string {
	t.Helper()

	tokenDir := filepath.Join(dataDir, "guild-1", "tokens", tokOwnerID)
	require.NoError(t, os.MkdirAll(tokenDir, 0o755))
	tokenPath := filepath.Join(tokenDir, "out.png")
	require.NoError(t, os.WriteFile(tokenPath, []byte("VICTIM_TOKEN_BYTES"), 0o644))

	_, err := database.NewInsert().Model(&models.Media{
		ID:        tokVictimMID,
		OwnerID:   tokOwnerID,
		Path:      tokenPath,
		URL:       "http://cdn/guild-1/tokens/" + tokOwnerID + "/out.png",
		Kind:      models.KindTokenPlayer,
		Name:      "victim",
		CreatedAt: time.Now(),
	}).Exec(context.Background())
	require.NoError(t, err)
	return tokenPath
}

func buildComponentInteraction(userID, customID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			User: &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      customID,
				ComponentType: discordgo.ButtonComponent,
			},
		},
	}
}

func buildSelectInteraction(userID, customID string, values []string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			User: &discordgo.User{ID: userID},
			Data: discordgo.MessageComponentInteractionData{
				CustomID:      customID,
				ComponentType: discordgo.SelectMenuComponent,
				Values:        values,
			},
		},
	}
}

/*
Attacker downloads someone else's token.

When:

	An attacker clicks a download button whose CustomID embeds the victim's
	mediaID (forged or scraped).

Expected:

	No file attachment is sent. The handler must reject the click before
	opening media.Path.

Development Note (v0.12.6, 20260511):

	Currently FAILS — handler reads the file and replies with the bytes.
*/
func TestTokenDownload_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenDownloadHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenDownloadPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokAttackerID, customID))

	/*
		The interaction response is sent via REST POST. Inspect bodies for the
		victim's bytes (base64 or raw won't appear here:
		discordgo serializes files into multipart).

		Easiest assertion: at least one captured request must NOT contain a
		Files payload section.

		Stronger: no request body should reference the victim filename.
	*/
	for _, r := range stub.Requests() {
		assert.NotContains(t, string(r.Body), "VICTIM_TOKEN_BYTES",
			"attacker received victim's token bytes via %s %s", r.Method, r.URL)
		assert.NotContains(t, string(r.Body), `"filename":"victim`,
			"attacker received victim's token attachment via %s %s", r.Method, r.URL)
	}
}

/*
Owner can download their own token (regression guard for the fix).

When:

	The legitimate owner clicks the download button for their own media.

Expected:

	A response is sent (we don't pin the exact shape: the file plumbing goes
	through discordgo's multipart writer). The handler must NOT reject this
	case once ownership checks are added.
*/
func TestTokenDownload_AllowsOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenDownloadHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenDownloadPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokOwnerID, customID))

	require.NotEmpty(t, stub.Requests(), "owner click must produce some Discord response")
}

/*
Attacker cannot delete someone else's token.

When:

	An attacker submits the delete-confirm interaction with the victim's
	mediaID.

Expected:

	The Media row remains in the DB and the file remains on disk.

Development Note (v0.12.6, 20260511):

	Currently FAILS: the handler issues DELETE and os.Remove with no ownership check.
*/
func TestTokenDeleteConfirm_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	tokenPath := seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenDeleteConfirmHandler{db: database, dataDir: dataDir}

	customID := fmt.Sprintf("%s:%s", messages.TokenDeleteConfirmPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokAttackerID, customID))

	var remaining models.Media
	err := database.NewSelect().Model(&remaining).Where("id = ?", tokVictimMID).Scan(context.Background())
	assert.NoError(t, err, "victim's media row must survive an attacker's delete")

	_, statErr := os.Stat(tokenPath)
	assert.NoError(t, statErr, "victim's token file must survive an attacker's delete")
}

/*
Attacker cannot assign someone else's token to their own campaign.

When:

	An attacker submits the assign-select interaction with the victim's
	mediaID and a campaign the attacker is a member of.

Expected:

	No CampaignPlayer row anywhere has its media_id set to the victim's
	mediaID.

Development Note (v0.12.6, 20260511):

	Currently FAILS: the handler runs the UPDATE without checking
	ownership of the media.
*/
func TestTokenGalleryAssignSelect_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	ctx := context.Background()
	const attackerCampaign = "camp-attacker"
	_, err := database.NewInsert().Model(&models.Campaign{
		ID:            attackerCampaign,
		Name:          "Attacker Campaign",
		Tag:           "atk",
		DungeonMaster: tokAttackerID,
		IsApproved:    true,
	}).Exec(ctx)
	require.NoError(t, err)
	_, err = database.NewInsert().Model(&models.CampaignPlayer{
		PlayerID:   tokAttackerID,
		CampaignID: attackerCampaign,
		Role:       models.RolePlayer,
		Status:     models.StatusActive,
	}).Exec(ctx)
	require.NoError(t, err)

	stub := testutil.NewStubSession(t)
	handler := &tokenGalleryAssignSelectHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenGalleryAssignSelectPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session,
		buildSelectInteraction(tokAttackerID, customID, []string{attackerCampaign}))

	count, err := database.NewSelect().Model((*models.CampaignPlayer)(nil)).
		Where("media_id = ?", tokVictimMID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count,
		"attacker assigned victim's token to a campaign — ownership check missing")
}

/*
Gallery detail view leaks the victim's token preview to a non-owner.

When:

	An attacker submits the gallery select with the victim's mediaID.

Expected:

	No response embed contains the victim's token URL.

Development Note (v0.12.6, 20260511):

	Currently FAILS: handler renders the embed (with image URL) for any caller.
*/
func TestTokenGallerySelect_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenGallerySelectHandler{db: database}

	handler.HandleComponents(stub.Session, buildSelectInteraction(
		tokAttackerID, messages.TokenGallerySelectPrefix, []string{tokVictimMID}))

	for _, r := range stub.Requests() {
		assert.NotContains(t, string(r.Body), "tokens/"+tokOwnerID,
			"attacker received victim's token URL in gallery detail response")
	}
}

/*
Delete-prompt screen leaks the victim's token name to a non-owner.

When:

	An attacker clicks a delete-prompt button with the victim's mediaID.

Expected:

	No response references the victim's media.

Development Note (v0.12.6, 20260511):

	Currently FAILS: handler renders the prompt for any caller.
*/
func TestTokenDeletePrompt_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenDeletePromptHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenDeletePromptPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokAttackerID, customID))

	for _, r := range stub.Requests() {
		assert.NotContains(t, string(r.Body), "victim",
			"attacker saw victim's token name in delete-prompt response")
	}
}

/*
Assign picker leaks the victim's token name to a non-owner.

When:

	An attacker clicks an assign button with the victim's mediaID.

Expected:

	No response references the victim's token.

Development Note (v0.12.6, 20260511):

	Currently FAILS.
*/
func TestTokenGalleryAssign_RejectsNonOwner(t *testing.T) {
	database := testutil.NewTestDB(t)
	dataDir := t.TempDir()
	seedVictimToken(t, database, dataDir)

	stub := testutil.NewStubSession(t)
	handler := &tokenGalleryAssignHandler{db: database}

	customID := fmt.Sprintf("%s:%s", messages.TokenGalleryAssignPrefix, tokVictimMID)
	handler.HandleComponents(stub.Session, buildComponentInteraction(tokAttackerID, customID))

	for _, r := range stub.Requests() {
		assert.NotContains(t, string(r.Body), "victim",
			"attacker saw victim's token name in assign-picker response")
	}
}
