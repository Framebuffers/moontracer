package interactions

/*
	Campaign channel + thread creation helpers.

	Called during campaign approval (campaign_approve.go) to set up the
	Discord category, text channel, and standard threads for a new campaign.
*/

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// defaultArchiveDuration is the auto-archive duration for new threads, in minutes (1 week).
const defaultArchiveDuration = 10080

// standardThreads are the threads auto-created in every approved campaign's channel.
var standardThreads = []string{"welcome", "announcements", "sessions", "dice-rolls", "general"}

// threadInitMessages is the pinned welcome message sent to each standard thread on creation.
var threadInitMessages = map[string]string{
	"announcements": messages.ThreadInitMsgAnnouncements,
	"sessions":      messages.ThreadInitMsgSessions,
	"dice-rolls":    messages.ThreadInitMsgDiceRolls,
	"general":       messages.ThreadInitMsgGeneral,
}

/*
EnsureCampaignRole creates the campaign's Discord role (named after the tag) and assigns it
to the DM. Idempotent: if RoleID is already set it skips creation.

Exported so the approval handler can call it independently of channel setup.
*/
func EnsureCampaignRole(s *discordgo.Session, guildID string, c *models.Campaign) error {
	if c.RoleID != "" {
		return nil
	}
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}
	role, err := guard.GuildRoleCreate(s, guildID, &discordgo.RoleParams{Name: channelName})
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	c.RoleID = role.ID
	if err := guard.GuildMemberRoleAdd(s, guildID, c.DungeonMaster, role.ID); err != nil {
		log.Printf("campaign_threads: assign role to DM %s: %v", c.DungeonMaster, err)
	}
	return nil
}

/*
SetupNewChannel creates a private text channel for the campaign under the shared Campaigns
category, then creates the standard threads inside it.

Mutates campaign in-place with CategoryID, ChannelID, AnnouncementsThreadID.
*/
func SetupNewChannel(s *discordgo.Session, guildID string, c *models.Campaign) error {
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}

	// Ensure role exists before setting channel permissions.
	if err := EnsureCampaignRole(s, guildID, c); err != nil {
		log.Printf("campaign_threads: role for new channel: %v", err)
	}

	categoryID, err := findOrCreateCampaignsCategory(s, guildID)
	if err != nil {
		return fmt.Errorf("resolve category: %w", err)
	}
	c.CategoryID = categoryID

	overwrites := []*discordgo.PermissionOverwrite{
		{
			ID:   guildID,
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		{
			ID:    s.State.User.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionManageThreads | discordgo.PermissionSendMessages,
		},
	}
	if c.RoleID != "" {
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID:    c.RoleID,
			Type:  discordgo.PermissionOverwriteTypeRole,
			Allow: discordgo.PermissionViewChannel,
		})
	}

	ch, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name:                 channelName,
		Type:                 discordgo.ChannelTypeGuildText,
		ParentID:             categoryID,
		PermissionOverwrites: overwrites,
	})
	if err != nil {
		return fmt.Errorf("create channel: %w", err)
	}
	c.ChannelID = ch.ID
	createStandardThreads(s, c, ch.ID, channelName)
	return nil
}

/*
SetupExistingChannel links the campaign to an already-existing Discord channel and creates
the standard threads inside it.

Mutates campaign in-place with ChannelID and AnnouncementsThreadID.
*/
func SetupExistingChannel(s *discordgo.Session, c *models.Campaign, channelID string) {
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}
	c.ChannelID = channelID
	createStandardThreads(s, c, channelID, channelName)
}

// createStandardThreads creates the standard threads in channelID and sets AnnouncementsThreadID.
func createStandardThreads(s *discordgo.Session, c *models.Campaign, channelID, channelName string) {
	for _, name := range standardThreads {
		threadName := fmt.Sprintf("%s-%s", channelName, name)
		thread, err := guard.ThreadStart(s, channelID, threadName, defaultArchiveDuration)
		if err != nil {
			log.Printf("campaign_threads: create thread %s: %v", threadName, err)
			continue
		}
		if name == "announcements" {
			c.AnnouncementsThreadID = thread.ID
		}

		// Resolve init message: welcome uses the campaign name; others use the static map.
		var initMsg string
		if name == "welcome" {
			initMsg = fmt.Sprintf(messages.ThreadInitMsgWelcomeFmt, c.Name)
		} else if msg, ok := threadInitMessages[name]; ok {
			initMsg = msg
		}

		if initMsg != "" {
			msg, err := guard.ChannelMessageSend(s, thread.ID, initMsg)
			if err != nil {
				log.Printf("campaign_threads: send init message to %s: %v", threadName, err)
				continue
			}
			if err := guard.ChannelMessagePin(s, thread.ID, msg.ID); err != nil {
				log.Printf("campaign_threads: pin init message in %s: %v", threadName, err)
			}
			if name == "welcome" {
				if err := guard.LockThread(s, thread.ID); err != nil {
					log.Printf("campaign_threads: lock welcome thread %s: %v", threadName, err)
				}
			}
		}
	}
}

// createCampaignChannels is kept for backward compatibility with any remaining call sites.
// New code should call EnsureCampaignRole + SetupNewChannel separately.
func createCampaignChannels(s *discordgo.Session, guildID string, c *models.Campaign) {
	if err := EnsureCampaignRole(s, guildID, c); err != nil {
		log.Printf("campaign_threads: %v", err)
		return
	}
	if err := SetupNewChannel(s, guildID, c); err != nil {
		log.Printf("campaign_threads: %v", err)
	}
}

/*
findOrCreateCampaignsCategory returns the ID of the shared "Campaigns" Discord category,
creating it if it doesn't already exist in the guild.
*/
func findOrCreateCampaignsCategory(s *discordgo.Session, guildID string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", fmt.Errorf("fetch guild channels: %w", err)
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildCategory &&
			strings.EqualFold(ch.Name, messages.CampaignsCategoryName) {
			return ch.ID, nil
		}
	}
	cat, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name: messages.CampaignsCategoryName,
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		return "", fmt.Errorf("create campaigns category: %w", err)
	}
	return cat.ID, nil
}

/*
RetireChannel locks down a campaign's Discord channel when the campaign is archived or deleted.

Permission changes applied to the channel:
  - @everyone: deny ViewChannel
  - Campaign role (if set): deny ViewChannel
  - Mod/admin Discord roles (from ADMIN_ROLE_NAME / MOD_ROLE_NAME env): allow ViewChannel

This makes the channel invisible to regular members while keeping it accessible to staff.
Errors are logged but non-fatal; the DB operation that triggered retirement already succeeded.
*/
func RetireChannel(s *discordgo.Session, guildID string, campaign *models.Campaign) {
	channelID := campaign.ChannelID
	if channelID == "" {
		return
	}

	// Resolve mod/admin Discord role IDs from env-configured names.
	adminRoleName := os.Getenv("ADMIN_ROLE_NAME")
	modRoleName := os.Getenv("MOD_ROLE_NAME")

	var staffRoleIDs []string
	if adminRoleName != "" || modRoleName != "" {
		roles, err := s.GuildRoles(guildID)
		if err != nil {
			log.Printf("campaign_threads: retire channel %s: fetch roles: %v", channelID, err)
		} else {
			for _, r := range roles {
				if (adminRoleName != "" && strings.EqualFold(r.Name, adminRoleName)) ||
					(modRoleName != "" && strings.EqualFold(r.Name, modRoleName)) {
					staffRoleIDs = append(staffRoleIDs, r.ID)
				}
			}
		}
	}

	// Deny @everyone.
	if err := guard.ChannelPermissionSet(s, channelID, guildID,
		discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionViewChannel); err != nil {
		log.Printf("campaign_threads: retire channel %s: deny everyone: %v", channelID, err)
	}

	// Deny campaign role.
	if campaign.RoleID != "" {
		if err := guard.ChannelPermissionSet(s, channelID, campaign.RoleID,
			discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionViewChannel); err != nil {
			log.Printf("campaign_threads: retire channel %s: deny campaign role: %v", channelID, err)
		}
	}

	// Allow staff roles.
	for _, roleID := range staffRoleIDs {
		if err := guard.ChannelPermissionSet(s, channelID, roleID,
			discordgo.PermissionOverwriteTypeRole, discordgo.PermissionViewChannel, 0); err != nil {
			log.Printf("campaign_threads: retire channel %s: allow staff role %s: %v", channelID, roleID, err)
		}
	}
}
