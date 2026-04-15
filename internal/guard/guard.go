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
DevMode is true when DEV_MODE=true is set.

In dev mode, debug-only UI surfaces are exposed: the Diagnostics button on
/admin, the /campaigndatabase slash command, and any future internal-tooling
buttons.

When false (the default, for production), those surfaces are hidden
and their handlers refuse to execute even if a stale CustomID is clicked.

Unrelated to SafeMode:

	You can run production with DevMode off AND SafeMode off
	(live mutations, hidden debug UI) or dev with both on.

	Staging may set SafeMode off + DevMode on to rehearse live mutations with debug visibility.
*/
var DevMode = os.Getenv("DEV_MODE") == "true"

/*
DebugAdminID is the Discord user ID granted admin privileges for testing.

The user must still be registered and not banned:
this only elevates their server role.
It does not bypass any other security checks.
*/
var DebugAdminID = os.Getenv("DEBUG_ADMIN_ID")

/*
DebugGuildID is the Discord guild ID the bot is scoped to in dev mode.

When DevMode is on and this is set, the handler refuses interactions from
any other guild. Sourced from DISCORD_GUILD_ID, the same env that scopes
command registration to a single guild in dev.
*/
var DebugGuildID = os.Getenv("DISCORD_GUILD_ID")

func init() {
	if SafeMode {
		log.Println("guard: SAFE_MODE is ON — Discord-mutating operations will be logged but not executed")
	}
	if DevMode {
		log.Println("guard: DEV_MODE is ON — debug UI surfaces (Diagnostics, /campaigndatabase) are visible")
		if DebugGuildID != "" {
			log.Printf("guard: DEV_MODE scoped to guild %s — interactions from other guilds will be rejected", DebugGuildID)
		}
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

// GuildChannelCreateComplex creates a guild channel, or logs and returns a fake channel in safe mode.
func GuildChannelCreateComplex(s *discordgo.Session, guildID string, data discordgo.GuildChannelCreateData) (*discordgo.Channel, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would create channel %q (type %d) in guild %s", data.Name, data.Type, guildID)
		return &discordgo.Channel{ID: "safe-mode-channel", Name: data.Name, Type: data.Type, ParentID: data.ParentID}, nil
	}
	return s.GuildChannelCreateComplex(guildID, data)
}

// ChannelDelete deletes a channel, or logs in safe mode.
func ChannelDelete(s *discordgo.Session, channelID string) (*discordgo.Channel, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would delete channel %s", channelID)
		return &discordgo.Channel{ID: channelID}, nil
	}
	return s.ChannelDelete(channelID)
}

// ThreadStart starts a new thread on a channel (no parent message), or logs in safe mode.
func ThreadStart(s *discordgo.Session, channelID, name string, archiveDuration int) (*discordgo.Channel, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would start thread %q in channel %s", name, channelID)
		return &discordgo.Channel{ID: "safe-mode-thread-" + name, Name: name}, nil
	}
	return s.ThreadStart(channelID, name, discordgo.ChannelTypeGuildPublicThread, archiveDuration)
}
