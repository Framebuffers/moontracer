package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Unit Testing: Campaign model methods.
*/

/*
PlayerMap returns all campaign players keyed by ID.

When:

	Campaign has three players (two regular, one DM).

Expected:

	Map contains all three, keyed by PlayerID.
*/
func TestPlayerMap(t *testing.T) {
	c := &Campaign{
		CampaignPlayers: []CampaignPlayer{
			{PlayerID: "p1", Role: RolePlayer},
			{PlayerID: "p2", Role: RoleDM},
			{PlayerID: "p3", Role: RolePlayer},
		},
	}

	m := c.PlayerMap()
	assert.Len(t, m, 3)
	assert.Equal(t, "p1", m["p1"].PlayerID)
	assert.Equal(t, "p2", m["p2"].PlayerID)
	assert.Equal(t, "p3", m["p3"].PlayerID)
}

/*
PlayerMap with no players.

When:

	Campaign has no CampaignPlayers loaded.

Expected:

	Returns an empty map.
*/
func TestPlayerMap_Empty(t *testing.T) {
	c := &Campaign{}
	m := c.PlayerMap()
	assert.Empty(t, m)
}

/*
DMMap returns only DM-role players keyed by ID.

When:

	Campaign has two DMs and two regular players.

Expected:

	Map contains only the two DMs. Regular players are excluded.
*/
func TestDMMap(t *testing.T) {
	c := &Campaign{
		CampaignPlayers: []CampaignPlayer{
			{PlayerID: "p1", Role: RolePlayer},
			{PlayerID: "p2", Role: RoleDM},
			{PlayerID: "p3", Role: RoleDM},
			{PlayerID: "p4", Role: RolePlayer},
		},
	}

	m := c.DMMap()
	assert.Len(t, m, 2)
	assert.Contains(t, m, "p2")
	assert.Contains(t, m, "p3")
	assert.NotContains(t, m, "p1")
	assert.NotContains(t, m, "p4")
}

/*
DMMap with no DMs.

When:

	Campaign has only regular players, no DM-role entries.

Expected:

	Returns an empty map.
*/
func TestDMMap_NoDMs(t *testing.T) {
	c := &Campaign{
		CampaignPlayers: []CampaignPlayer{
			{PlayerID: "p1", Role: RolePlayer},
		},
	}

	m := c.DMMap()
	assert.Empty(t, m)
}
