package interactions

/*
	Campaign channel + thread creation helpers.

	Called during campaign approval (campaign_approve.go) to set up the
	Discord category, text channel, and standard threads for a new campaign.
*/

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
)

// defaultArchiveDuration is the auto-archive duration for new threads, in minutes (1 week).
const defaultArchiveDuration = 10080

// standardThreads are the threads auto-created in every approved campaign's channel.
var standardThreads = []string{"announcements", "sessions", "general"}

/*
createCampaignChannels creates a category, text channel, and standard threads for a campaign.

On success, mutates the campaign in-place with CategoryID, ChannelID, and AnnouncementsThreadID.
Errors are logged but non-fatal — partial setup is better than none.
*/
func createCampaignChannels(s *discordgo.Session, guildID string, c *models.Campaign) {
	// Category.
	category, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name: c.Name,
		Type: discordgo.ChannelTypeGuildCategory,
	})
	if err != nil {
		log.Printf("campaign_threads: failed to create category for %s: %v", c.ID, err)
		return
	}
	c.CategoryID = category.ID

	// Text channel under category.
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}
	ch, err := guard.GuildChannelCreateComplex(s, guildID, discordgo.GuildChannelCreateData{
		Name:     channelName,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID,
	})
	if err != nil {
		log.Printf("campaign_threads: failed to create channel for %s: %v", c.ID, err)
		return
	}
	c.ChannelID = ch.ID

	// Standard threads.
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
