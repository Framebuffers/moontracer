package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Unit Testing: Player model methods.
*/

/*
IsAdmin truth table.

When:

	Player role is Player, Mod, or Admin.

Expected:

	Only Admin returns true.
*/
func TestIsAdmin(t *testing.T) {
	tests := []struct {
		role ServerRole
		want bool
	}{
		{ServerRolePlayer, false},
		{ServerRoleMod, false},
		{ServerRoleAdmin, true},
	}
	for _, tt := range tests {
		p := &Player{Role: tt.role}
		assert.Equal(t, tt.want, p.IsAdmin(), "IsAdmin() for role %q", tt.role)
	}
}

/*
IsMod truth table.

When:

	Player role is Player, Mod, or Admin.

Expected:

	Mod and Admin return true. Admin implies mod.
*/
func TestIsMod(t *testing.T) {
	tests := []struct {
		role ServerRole
		want bool
	}{
		{ServerRolePlayer, false},
		{ServerRoleMod, true},
		{ServerRoleAdmin, true},
	}
	for _, tt := range tests {
		p := &Player{Role: tt.role}
		assert.Equal(t, tt.want, p.IsMod(), "IsMod() for role %q", tt.role)
	}
}

/*
IsDMOf checks campaign-scoped DM ownership.

When:

	Player is DM of camp-a, regular player in camp-b.

Expected:

	True for camp-a, false for camp-b, false for camp-c (not in campaign at all).
*/
func TestIsDMOf(t *testing.T) {
	p := &Player{
		CampaignPlayers: []CampaignPlayer{
			{CampaignID: "camp-a", Role: RoleDM},
			{CampaignID: "camp-b", Role: RolePlayer},
		},
	}

	assert.True(t, p.IsDMOf("camp-a"), "should be DM of camp-a")
	assert.False(t, p.IsDMOf("camp-b"), "player role in camp-b, not DM")
	assert.False(t, p.IsDMOf("camp-c"), "not in camp-c at all")
}

/*
IsMemberOf checks active campaign membership.

When:

	Player is active in camp-a, on hiatus in camp-b, banned in camp-c.

Expected:

	Only active status returns true. Hiatus, banned, and absent all return false.
*/
func TestIsMemberOf(t *testing.T) {
	p := &Player{
		CampaignPlayers: []CampaignPlayer{
			{CampaignID: "camp-a", Status: StatusActive},
			{CampaignID: "camp-b", Status: StatusHiatus},
			{CampaignID: "camp-c", Status: StatusBanned},
		},
	}

	assert.True(t, p.IsMemberOf("camp-a"), "active member")
	assert.False(t, p.IsMemberOf("camp-b"), "hiatus is not active")
	assert.False(t, p.IsMemberOf("camp-c"), "banned is not active")
	assert.False(t, p.IsMemberOf("camp-d"), "not in campaign")
}

/*
DMCampaignIDs returns IDs of all campaigns the player DMs.

When:

	Player is DM of camp-a and camp-c, regular player in camp-b.

Expected:

	Returns ["camp-a", "camp-c"]. Order does not matter.
*/
func TestDMCampaignIDs(t *testing.T) {
	p := &Player{
		CampaignPlayers: []CampaignPlayer{
			{CampaignID: "camp-a", Role: RoleDM},
			{CampaignID: "camp-b", Role: RolePlayer},
			{CampaignID: "camp-c", Role: RoleDM},
		},
	}

	ids := p.DMCampaignIDs()
	assert.ElementsMatch(t, []string{"camp-a", "camp-c"}, ids)
}

/*
DMCampaignIDs with no campaigns.

When:

	Player has no CampaignPlayers loaded.

Expected:

	Returns nil.
*/
func TestDMCampaignIDs_Empty(t *testing.T) {
	p := &Player{}
	assert.Nil(t, p.DMCampaignIDs(), "no campaigns → nil")
}
