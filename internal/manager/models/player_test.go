package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestDMCampaignIDs_Empty(t *testing.T) {
	p := &Player{}
	assert.Nil(t, p.DMCampaignIDs(), "no campaigns → nil")
}
