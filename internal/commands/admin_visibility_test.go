package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/messages"
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
allHubCustomIDs returns CustomIDs of every button across all admin hub rows.
*/
func allHubCustomIDs(t *testing.T) []string {
	t.Helper()
	data := adminHubData()
	var ids []string
	for _, comp := range data.Components {
		row, ok := comp.(discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range row.Components {
			btn, ok := c.(discordgo.Button)
			if !ok {
				continue
			}
			ids = append(ids, btn.CustomID)
		}
	}
	return ids
}

/*
DevMode on exposes the Diagnostics button.

Layout (DevMode=true): row0=[Query Campaigns, Broadcast, Settings], row1=[Diag], row2=[Back].
*/
func TestAdminHub_DevModeShowsDebugButtons(t *testing.T) {
	guard.SetModesForTest(t, true, true)

	ids := allHubCustomIDs(t)
	assert.Contains(t, ids, messages.AdminDiagPrefix, "DevMode hub should expose Diagnostics")
	assert.Contains(t, ids, messages.AdminSettingsPrefix, "Settings must always be present")
}

/*
DevMode off hides Diagnostics.

Layout (DevMode=false): row0=[Query Campaigns, Broadcast, Settings], row1=[Back].
No debug surfaces exposed to real users.
*/
func TestAdminHub_NonDevModeHidesDebugButtons(t *testing.T) {
	guard.SetModesForTest(t, true, false)

	ids := allHubCustomIDs(t)
	assert.Contains(t, ids, messages.AdminSettingsPrefix, "Settings must always be present")
	assert.NotContains(t, ids, messages.AdminDatabasePrefix, "Database button must not leak in production")
	assert.NotContains(t, ids, messages.AdminDiagPrefix, "Diagnostics button must not leak in production")
}
