package auth

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
)

// Scope represents a permission level required to perform an action.
// Scopes are independent — each is checked against its own data source.
type Scope string

const (
	// ScopePlayer — any registered player (exists in players table).
	ScopePlayer Scope = "player"

	// ScopeMember — an active participant in a specific campaign.
	ScopeMember Scope = "member"

	// ScopeDM — the dungeon master of a specific campaign.
	ScopeDM Scope = "dm"

	// ScopeMod — a user with the admin/mod Discord role in the guild.
	ScopeMod Scope = "mod"
)

// TODO(human): Implement the Authorize function.
// Authorize checks whether a user has the required scope.
// For ScopePlayer and ScopeMod, campaignID can be empty.
// For ScopeMember and ScopeDM, campaignID is required.
func Authorize(database *bun.DB, s *discordgo.Session, userID, guildID, adminRole string, required Scope, campaignID string) (bool, error) {
	return false, nil
}

// Resolve returns all user IDs that match the given scope for a campaign.
// Used by the notification system to determine who should receive a message.
func Resolve(database *bun.DB, s *discordgo.Session, campaignID, guildID, adminRole string, scope Scope) ([]string, error) {
	switch scope {
	case ScopePlayer:
		players, err := db.GetAll[models.Player](database)
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(players))
		for i, p := range players {
			ids[i] = p.ID
		}
		return ids, nil

	case ScopeMember:
		players, err := models.GetCampaignPlayers(database, campaignID)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, p := range players {
			if p.Status == models.StatusActive {
				ids = append(ids, p.PlayerID)
			}
		}
		return ids, nil

	case ScopeDM:
		campaign, err := db.GetByID[models.Campaign](database, campaignID)
		if err != nil {
			return nil, err
		}
		return []string{campaign.DungeonMaster}, nil

	case ScopeMod:
		return adminsWithRole(s, guildID, adminRole)

	default:
		return nil, nil
	}
}

// adminsWithRole returns user IDs of guild members who have the given role name.
func adminsWithRole(s *discordgo.Session, guildID, roleName string) ([]string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	var roleID string
	for _, r := range roles {
		if r.Name == roleName {
			roleID = r.ID
			break
		}
	}
	if roleID == "" {
		return nil, nil
	}

	var ids []string
	after := ""
	for {
		members, err := s.GuildMembers(guildID, after, 1000)
		if err != nil {
			return nil, err
		}
		if len(members) == 0 {
			break
		}
		for _, m := range members {
			for _, r := range m.Roles {
				if r == roleID {
					ids = append(ids, m.User.ID)
					break
				}
			}
		}
		after = members[len(members)-1].User.ID
		if len(members) < 1000 {
			break
		}
	}
	return ids, nil
}
