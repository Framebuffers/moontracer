package interactions

/*
	Campaign channel + thread creation helpers.

	Called during campaign approval (campaign_approve.go) to set up the
	Discord category, text channel, and standard threads for a new campaign.
*/

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// defaultArchiveDuration is the auto-archive duration for new threads, in minutes (1 week).
const defaultArchiveDuration = 10080

// standardThreads are the threads auto-created in every approved campaign's channel.
var standardThreads = []string{"announcements", "sessions", "general"}

/*
createCampaignChannels creates a role, a private text channel, and standard threads for a campaign,
grouped under a shared "Campaigns" category (found or created).

The channel is visible only to members of the campaign role (@everyone is denied ViewChannel).
The DM is immediately assigned the role.

On success, mutates the campaign in-place with RoleID, CategoryID, ChannelID, and AnnouncementsThreadID.
Errors are logged but non-fatal — partial setup is better than none.
*/
func createCampaignChannels(s *discordgo.Session, guildID string, c *models.Campaign) {
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}

	role, err := guard.GuildRoleCreate(s, guildID, &discordgo.RoleParams{
		Name: c.Name,
	})
	if err != nil {
		log.Printf("campaign_threads: failed to create role for %s: %v", c.ID, err)
		return
	}
	c.RoleID = role.ID

	if err := guard.GuildMemberRoleAdd(s, guildID, c.DungeonMaster, role.ID); err != nil {
		log.Printf("campaign_threads: failed to assign role to DM for %s: %v", c.ID, err)
	}

	categoryID, err := findOrCreateCampaignsCategory(s, guildID)
	if err != nil {
		log.Printf("campaign_threads: failed to resolve campaigns category: %v", err)
		return
	}
	c.CategoryID = categoryID

	ch, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: categoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				// Deny @everyone view access (everyone role ID == guild ID in Discord).
				ID:   guildID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
			{
				ID:    role.ID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel,
			},
		},
	})
	if err != nil {
		log.Printf("campaign_threads: failed to create channel for %s: %v", c.ID, err)
		return
	}
	c.ChannelID = ch.ID

	for _, name := range standardThreads {
		threadName := fmt.Sprintf("%s-%s", channelName, name)
		thread, err := guard.ThreadStart(s, ch.ID, threadName, defaultArchiveDuration)
		if err != nil {
			log.Printf("campaign_threads: failed to create thread %s: %v", threadName, err)
			continue
		}
		if name == "announcements" {
			c.AnnouncementsThreadID = thread.ID
		}
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
