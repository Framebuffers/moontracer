package guard

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

/*
SafeMode is true when SAFE_MODE=true is set.

In safe mode, all Discord-mutating operations (role create/add/remove, DMs)
are logged but not executed.

DB operations still run normally so the bot can be tested end-to-end without impacting the server.
*/
var SafeMode = os.Getenv("SAFE_MODE") == "true"

/*
DebugAdminID is the Discord user ID granted admin privileges for testing.

The user must still be registered and not banned:
this only elevates their server role.
It does not bypass any other security checks.
*/
var DebugAdminID = os.Getenv("DEBUG_ADMIN_ID")

func init() {
	if SafeMode {
		log.Println("guard: SAFE_MODE is ON — Discord-mutating operations will be logged but not executed")
	}
	if DebugAdminID != "" {
		log.Printf("guard: DEBUG_ADMIN_ID is set — user %s will be treated as admin", DebugAdminID)
		if !SafeMode {
			log.Println("guard: WARNING — DEBUG_ADMIN_ID is set with SAFE_MODE OFF; elevation requires the Discord admin role as confirmation")
		}
	}
}

// GuildRoleCreate creates a guild role, or logs and returns a fake role in safe mode.
func GuildRoleCreate(s *discordgo.Session, guildID string, params *discordgo.RoleParams) (*discordgo.Role, error) {
	if SafeMode {
		name := ""
		if params != nil {
			name = params.Name
		}
		log.Printf("guard: [SAFE_MODE] would create role %q in guild %s", name, guildID)
		return &discordgo.Role{ID: "safe-mode-role", Name: name}, nil
	}
	return s.GuildRoleCreate(guildID, params)
}

// GuildMemberRoleAdd adds a role to a member, or logs in safe mode.
func GuildMemberRoleAdd(s *discordgo.Session, guildID, userID, roleID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would add role %s to user %s in guild %s", roleID, userID, guildID)
		return nil
	}
	return s.GuildMemberRoleAdd(guildID, userID, roleID)
}

// GuildMemberRoleRemove removes a role from a member, or logs in safe mode.
func GuildMemberRoleRemove(s *discordgo.Session, guildID, userID, roleID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would remove role %s from user %s in guild %s", roleID, userID, guildID)
		return nil
	}
	return s.GuildMemberRoleRemove(guildID, userID, roleID)
}
