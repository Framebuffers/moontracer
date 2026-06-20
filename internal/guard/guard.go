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
DevMode is true when DEV_MODE=true is set, OR when DISCORD_GUILD_ID is set.

Rationale:

	Setting DISCORD_GUILD_ID scopes the bot to a single test server,
	which is only done during development/staging. Treating that as dev mode
	means debug UI lights up automatically on that guild with no second flag.
	Production deployments leave DISCORD_GUILD_ID empty and DEV_MODE unset.

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
var DevMode = os.Getenv("DEV_MODE") == "true" || os.Getenv("DISCORD_GUILD_ID") != ""

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
		log.Println("guard: SAFE_MODE is ON- Discord-mutating operations will be logged but not executed")
	}
	if DevMode {
		log.Println("guard: DEV_MODE is ON- debug UI surfaces (Diagnostics, /campaigndatabase) are visible")
		if DebugGuildID != "" {
			log.Printf("guard: DEV_MODE scoped to guild %s- interactions from other guilds will be rejected", DebugGuildID)
		}
	}
	if DebugAdminID != "" {
		log.Printf("guard: DEBUG_ADMIN_ID is set- user %s will be treated as admin", DebugAdminID)
		if !SafeMode {
			log.Println("guard: WARNING- DEBUG_ADMIN_ID is set with SAFE_MODE OFF; elevation requires the Discord admin role as confirmation")
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

// GuildRoleEdit renames (or otherwise edits) an existing guild role, or logs in safe mode.
func GuildRoleEdit(s *discordgo.Session, guildID, roleID string, params *discordgo.RoleParams) (*discordgo.Role, error) {
	if SafeMode {
		name := ""
		if params != nil {
			name = params.Name
		}
		log.Printf("guard: [SAFE_MODE] would rename role %s to %q in guild %s", roleID, name, guildID)
		return &discordgo.Role{ID: roleID, Name: name}, nil
	}
	return s.GuildRoleEdit(guildID, roleID, params)
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

// ThreadCreate starts a new thread on a channel (no parent message), or logs in safe mode.
func ThreadCreate(s *discordgo.Session, channelID, name string, archiveDuration int) (*discordgo.Channel, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would start thread %q in channel %s", name, channelID)
		return &discordgo.Channel{ID: "safe-mode-thread-" + name, Name: name}, nil
	}
	return s.ThreadStart(channelID, name, discordgo.ChannelTypeGuildPublicThread, archiveDuration)
}

// ChannelMessageSend sends a message to a channel, or logs in safe mode.
func ChannelMessageSend(s *discordgo.Session, channelID, content string) (*discordgo.Message, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would send message to channel %s: %q", channelID, content)
		return &discordgo.Message{ID: "safe-mode-message"}, nil
	}
	return s.ChannelMessageSend(channelID, content)
}

// ChannelMessageSendComplex sends a rich message (embeds, components) to a channel, or logs in safe mode.
func ChannelMessageSendComplex(s *discordgo.Session, channelID string, data *discordgo.MessageSend) (*discordgo.Message, error) {
	if data.AllowedMentions == nil {
		data.AllowedMentions = &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
		}
	}

	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would send complex message to channel %s", channelID)
		return &discordgo.Message{ID: "safe-mode-message"}, nil
	}
	return s.ChannelMessageSendComplex(channelID, data)
}

// ChannelMessagePin pins a message in a channel, or logs in safe mode.
func ChannelMessagePin(s *discordgo.Session, channelID, messageID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would pin message %s in channel %s", messageID, channelID)
		return nil
	}
	return s.ChannelMessagePin(channelID, messageID)
}

// LockThread locks a thread so only members with Manage Threads can send messages in it, or logs in safe mode.
func LockThread(s *discordgo.Session, threadID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would lock thread %s", threadID)
		return nil
	}
	locked := true
	_, err := s.ChannelEdit(threadID, &discordgo.ChannelEdit{Locked: &locked})
	return err
}

// ChannelPermissionSet sets a permission overwrite on a channel, or logs in safe mode.
func ChannelPermissionSet(s *discordgo.Session, channelID, targetID string, targetType discordgo.PermissionOverwriteType, allow, deny int64) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would set permission on channel %s target %s allow=%d deny=%d", channelID, targetID, allow, deny)
		return nil
	}
	return s.ChannelPermissionSet(channelID, targetID, targetType, allow, deny)
}

// GuildRoleDelete deletes a Discord role, or logs in safe mode.
func GuildRoleDelete(s *discordgo.Session, guildID, roleID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would delete role %s in guild %s", roleID, guildID)
		return nil
	}
	return s.GuildRoleDelete(guildID, roleID)
}

// ThreadMemberAdd adds a user to a thread so it appears in their sidebar, or logs in safe mode.
func ThreadMemberAdd(s *discordgo.Session, threadID, userID string) error {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would add user %s to thread %s", userID, threadID)
		return nil
	}
	return s.ThreadMemberAdd(threadID, userID)
}

// ChannelMessageEdit edits an existing message in a channel, or logs in safe mode.
func ChannelMessageEdit(s *discordgo.Session, channelID, messageID, content string) (*discordgo.Message, error) {
	if SafeMode {
		log.Printf("guard: [SAFE_MODE] would edit message %s in channel %s", messageID, channelID)
		return &discordgo.Message{ID: messageID}, nil
	}
	return s.ChannelMessageEdit(channelID, messageID, content)
}
