package auth

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
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

/*
SyncServerRoles reads guild members from Discord and updates Player.Role
in the database to match.

Players with the admin Discord role get ServerRoleAdmin.
Players with the mod Discord role get ServerRoleMod.
All other registered players get ServerRolePlayer.

modRoleName is optional: if empty, mod is not synced from Discord.
Call this on bot startup and on GuildMemberUpdate events.
*/
func SyncServerRoles(database *bun.DB, s *discordgo.Session, guildID, adminRoleName, modRoleName string) error {
	adminIDs, err := adminsWithRole(s, guildID, adminRoleName)
	if err != nil {
		return err
	}

	var modIDs []string
	if modRoleName != "" {
		modIDs, err = adminsWithRole(s, guildID, modRoleName)
		if err != nil {
			return err
		}
	}

	return syncRoles(database, adminIDs, modIDs)
}

/*
syncRoles is the core sync logic, separated from the Discord API call
so it can be tested without a live session.

Notes:

	Admin > Mod > Player. A user in both sets gets Admin (higher wins).
	When DEBUG_ADMIN_ID is set, elevate the debug admin through the normal role path.

Design note (two-factor guard pattern):

	The DEBUG_ADMIN_ID block below is the first instance of a two-factor
	authorization: "env var claims X" + "Discord role confirms X". If a third
	source appears, consider extracting a composable guard:

		AuthorizeAll(db, userID, campaignID, ...func() (bool, error)) (bool, error)

	where each check is a closure over a different source (DB scope, Discord role,
	env var). Until then, inline is fine.
*/
func syncRoles(database *bun.DB, adminIDs, modIDs []string) error {
	ctx := context.Background()

	adminSet := make(map[string]bool, len(adminIDs))
	for _, id := range adminIDs {
		adminSet[id] = true
	}

	modSet := make(map[string]bool, len(modIDs))
	for _, id := range modIDs {
		modSet[id] = true
	}

	if guard.DebugAdminID != "" {
		if guard.SafeMode {
			// Safe mode: env var alone is sufficient for testing.
			if !adminSet[guard.DebugAdminID] {
				adminSet[guard.DebugAdminID] = true
				log.Printf("sync: debug admin %s injected into admin set (safe mode)", guard.DebugAdminID)
			}
		} else if adminSet[guard.DebugAdminID] {
			// Production: env var confirms the Discord role (two-factor).
			log.Printf("sync: debug admin %s confirmed by Discord role", guard.DebugAdminID)
		} else {
			log.Printf("sync: WARNING - debug admin %s does not have the Discord admin role; elevation denied", guard.DebugAdminID)
		}
	}

	var players []models.Player
	if err := database.NewSelect().Model(&players).Scan(ctx); err != nil {
		return err
	}

	for _, p := range players {
		var desired models.ServerRole
		if adminSet[p.ID] {
			desired = models.ServerRoleAdmin
		} else if modSet[p.ID] {
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
			log.Printf("sync: synced role for %s: %s -> %s", p.ID, p.Role, desired)
		}
	}

	return nil
}
