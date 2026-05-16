package auth

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
)

/*
	Flow:
		1. Loads Player.
		2. Checks if Player has a GlobalBan.
		3. Switches on scope: Player (if it exists and is not banned), Member, DM, Mod, Admin.
		4. Returns a bool if authorized or not.

	Note:
		In the case a composite query if two or more permissions are set, use `AuthorizeAny()`
*/

// Scope represents a permission level required to perform an action.
type Scope string

const (
	// ScopePlayer — any registered player (exists in players table, not banned).
	ScopePlayer Scope = "player"

	// ScopeMember — an active participant in a specific campaign.
	ScopeMember Scope = "member"

	// ScopeDM — the dungeon master of a specific campaign.
	ScopeDM Scope = "dm"

	// ScopeMod — a user with the mod or admin server role.
	ScopeMod Scope = "mod"

	// ScopeAdmin — a user with the admin server role.
	ScopeAdmin Scope = "admin"
)

/*
Authorize checks whether a user has the required scope.

- For ScopePlayer, ScopeMod, and ScopeAdmin, campaignID can be empty.

- For ScopeMember and ScopeDM, campaignID is required.

Authorize then loads the player in a single query, filtering CampaignPlayers to the
relevant campaign when needed.

Returns:

	true if authorized, false otherwise.
*/
func Authorize(database *bun.DB, userID string, required Scope, campaignID string) (bool, error) {
	ctx := context.Background()

	var player models.Player
	q := database.NewSelect().Model(&player).Where("id = ?", userID)

	// Only eager-load campaign memberships when the scope needs them.
	if required == ScopeMember || required == ScopeDM {
		q = q.Relation("CampaignPlayers", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("campaign_id = ?", campaignID)
		})
	}

	err := q.Scan(ctx)
	if err != nil {
		return false, nil // player not found = not authorized
	}

	// Global ban blocks everything.
	if player.PlayerIsBanned {
		return false, nil
	}

	switch required {
	case ScopePlayer:
		return true, nil // exists and not banned

	case ScopeMember:
		return player.IsMemberOf(campaignID), nil

	case ScopeDM:
		if player.IsDMOf(campaignID) {
			return true, nil
		}
		/*
			Fallback: check Campaign.DungeonMaster directly.
			Covers campaigns where the campaign_players row is missing (e.g. legacy data).
		*/
		campaign, err := db.GetByID[models.Campaign](database, campaignID)
		if err != nil {
			return false, nil
		}
		return campaign.DungeonMaster == userID, nil

	case ScopeMod:
		return player.IsMod(), nil

	case ScopeAdmin:
		return player.IsAdmin(), nil

	default:
		return false, fmt.Errorf("unknown scope: %s", required)
	}
}

// AuthorizeAny returns true if the user has ANY of the provided scopes.
// Useful for compound checks like "DM or Mod can do this", because: everyone is a Player.
func AuthorizeAny(database *bun.DB, userID string, campaignID string, scopes ...Scope) (bool, error) {
	for _, scope := range scopes {
		ok, err := Authorize(database, userID, scope, campaignID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// Resolve returns all user IDs that match the given scope for a campaign.
// Used by the notification system to determine who should receive a message.
func Resolve(database *bun.DB, s *discordgo.Session, campaignID, guildID, adminRole string, scope Scope) ([]string, error) {
	ctx := context.Background()

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

	case ScopeMod, ScopeAdmin:
		// Query the DB for players with the matching server role.
		var players []models.Player
		q := database.NewSelect().Model(&players)
		if scope == ScopeAdmin {
			q = q.Where("role = ?", models.ServerRoleAdmin)
		} else {
			q = q.Where("role IN (?)", bun.In([]models.ServerRole{models.ServerRoleMod, models.ServerRoleAdmin}))
		}
		if err := q.Scan(ctx); err != nil {
			return nil, err
		}
		ids := make([]string, len(players))
		for i, p := range players {
			ids[i] = p.ID
		}
		return ids, nil

	default:
		return nil, nil
	}
}

// adminsWithRole returns user IDs of guild members who have the given role name.
// Used by SyncServerRoles, not by Authorize (which uses the DB).
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
