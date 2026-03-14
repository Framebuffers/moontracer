package auth

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

/*
	Flow:
		1. Figure out which users have the Admin role.
		2. Fetch all registered members.
		3. Check if they have the roles they need for their permission level (admin, mod), else, gets "demoted" to Player.
		4. Update permissions.

	Notes:
		- What "DB-only" assignments mean, is a feature where an Admin can assign the role of a Moderator,
		within the context of this bot, without having them to be a server-wide mod.
		- This is useful when there are moderators that only have Campaign moderation access, enabling more granular permissions control.
*/

// SyncServerRoles reads guild members from Discord and updates Player.Role
// in the database to match.
//
// Players with the admin Discord role get ServerRoleAdmin;
// all other registered players keep ServerRolePlayer.
// Mod is reserved for future DB-only assignment by admins.
//
// Call on bot startup and on GuildMemberUpdate events.
func SyncServerRoles(database *bun.DB, s *discordgo.Session, guildID, adminRoleName string) error {
	ctx := context.Background()

	adminIDs, err := adminsWithRole(s, guildID, adminRoleName)
	if err != nil {
		return err
	}

	adminSet := make(map[string]bool, len(adminIDs))
	for _, id := range adminIDs {
		adminSet[id] = true
	}

	var players []models.Player
	if err := database.NewSelect().Model(&players).Scan(ctx); err != nil {
		return err
	}

	for _, p := range players {
		var desired models.ServerRole
		if adminSet[p.ID] {
			desired = models.ServerRoleAdmin
		} else if p.Role == models.ServerRoleMod {
			// Preserve DB-only mod assignments; don't demote mods to player.
			desired = models.ServerRoleMod
		} else {
			desired = models.ServerRolePlayer
		}

		if p.Role != desired {
			_, err := database.NewUpdate().Model((*models.Player)(nil)).
				Set("role = ?", desired).
				Where("id = ?", p.ID).
				Exec(ctx)
			if err != nil {
				return err
			}
			log.Printf("synced role for %s: %s -> %s", p.ID, p.Role, desired)
		}
	}

	return nil
}
