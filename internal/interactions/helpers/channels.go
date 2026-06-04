package helpers

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
FindOrCreateCampaignsCategory returns the ID of the shared "Campaigns" Discord category,
creating it if it doesn't already exist in the guild.
*/
func FindOrCreateCampaignsCategory(s *discordgo.Session, guildID string) (string, error) {
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
FindOrCreateForumChannel finds a forum channel with the given name inside categoryID,
or creates one if not.
*/
func FindOrCreateForumChannel(s *discordgo.Session, guildID, categoryID, name string) (string, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", fmt.Errorf("fetch guild channels: %w", err)
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildForum &&
			ch.ParentID == categoryID &&
			strings.EqualFold(ch.Name, name) {
			return ch.ID, nil
		}
	}
	ch, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildForum,
		ParentID: categoryID,
	})
	if err != nil {
		return "", fmt.Errorf("create forum channel %q: %w", name, err)
	}
	return ch.ID, nil
}

/*
FindForumChannel looks for a forum channel with the given name inside categoryID.

Returns ("", false) if not found.
*/
func FindForumChannel(s *discordgo.Session, guildID, categoryID, name string) (string, bool) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return "", false
	}
	for _, ch := range channels {
		if ch.Type == discordgo.ChannelTypeGuildForum &&
			ch.ParentID == categoryID &&
			strings.EqualFold(ch.Name, name) {
			return ch.ID, true
		}
	}
	return "", false
}

// BillboardChannelFromSettings returns the configured forum channel ID for c's format, or "".
func BillboardChannelFromSettings(s *models.GuildSettings, c *models.Campaign) string {
	if c.IsOneshot {
		return s.BillboardChannelOneshot
	}
	if c.IsWestmarch {
		return s.BillboardChannelWestmarch
	}
	return s.BillboardChannelCampaign
}
