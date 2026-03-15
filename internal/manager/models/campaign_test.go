package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestPlayerMap_Empty(t *testing.T) {
	c := &Campaign{}
	m := c.PlayerMap()
	assert.Empty(t, m)
}

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

func TestDMMap_NoDMs(t *testing.T) {
	c := &Campaign{
		CampaignPlayers: []CampaignPlayer{
			{PlayerID: "p1", Role: RolePlayer},
		},
	}

	m := c.DMMap()
	assert.Empty(t, m)
}
