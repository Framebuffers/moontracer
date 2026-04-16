package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"moontracer/internal/guard"
	"moontracer/internal/messages"
)

/*
Unit Testing
/admin hub button visibility (DevMode gating)

Covers adminHubData():
	The second action row conditionally includes the
	Database and Diagnostics buttons when guard.DevMode is true.

	Settings is always visible.

Note:
	This is the live-deploy guard rail that hides debug surfaces
	from real users when DEV_MODE is off.
*/

/*
secondRowCustomIDs returns the CustomIDs of every button in the admin hub's
second action row (the DevMode-gated row).
*/
func secondRowCustomIDs(t *testing.T) []string {
	t.Helper()
	data := adminHubData()
	if len(data.Components) < 2 {
		t.Fatalf("adminHubData should have at least 2 component rows, got %d", len(data.Components))
	}
	row, ok := data.Components[1].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("second component should be ActionsRow, got %T", data.Components[1])
	}

	ids := make([]string, 0, len(row.Components))
	for _, c := range row.Components {
		btn, ok := c.(discordgo.Button)
		if !ok {
			t.Fatalf("second-row component should be Button, got %T", c)
		}
		ids = append(ids, btn.CustomID)
	}
	return ids
}

/*
DevMode on exposes Database + Diagnostics.

When:

	guard.DevMode == true at hub-render time.

Expected:

	The second row contains Database, Settings, and Diagnostics buttons (in
	that order). Diag is the trailing button; Settings always sits in middle.
*/
func TestAdminHub_DevModeShowsDebugButtons(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	ids := secondRowCustomIDs(t)
	assert.Equal(t, []string{
		messages.AdminDatabasePrefix,
		messages.AdminSettingsPrefix,
		messages.AdminDiagPrefix,
	}, ids, "DevMode hub should expose Database + Settings + Diagnostics")
}

/*
DevMode off hides Database + Diagnostics.

When:

	guard.DevMode == false (real production deploy).

Expected:

	The second row contains only Settings. Database and Diag are stripped so
	no debug surface leaks to real users.
*/
func TestAdminHub_NonDevModeHidesDebugButtons(t *testing.T) {
	guard.SetModesForTest(t, true, false)

	ids := secondRowCustomIDs(t)
	assert.Equal(t, []string{messages.AdminSettingsPrefix}, ids, "non-DevMode hub should expose Settings only")
	assert.NotContains(t, ids, messages.AdminDatabasePrefix, "Database button must not leak in production")
	assert.NotContains(t, ids, messages.AdminDiagPrefix, "Diagnostics button must not leak in production")
}
