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
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// defaultArchiveDuration is the auto-archive duration for new threads, in minutes (1 week).
const defaultArchiveDuration = 10080

// standardThreads are the threads auto-created in every approved campaign's channel.
// Locked threads (welcome, announcements, sessions) are DM/bot-only; the rest are open to players.
var standardThreads = []string{"welcome", "announcements", "sessions", "dice-rolls", "characters", "memes", "art", "downtime", "resources"}

// threadInitMessages is the pinned welcome message sent to each standard thread on creation.
var threadInitMessages = map[string]string{
	"announcements": messages.ThreadInitMsgAnnouncements,
	"sessions":      messages.ThreadInitMsgSessions,
	"dice-rolls":    messages.ThreadInitMsgDiceRolls,
}

/*
EnsureCampaignRole creates the campaign's Discord role (named after the tag) and assigns it
to the DM. If RoleID is already set it skips creation.

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
func SetupNewChannel(database *bun.DB, s *discordgo.Session, guildID string, c *models.Campaign) error {
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name) // The fix for the weird characters inside names.
	}

	// Ensure role exists before setting channel permissions.
	if err := EnsureCampaignRole(s, guildID, c); err != nil {
		log.Printf("campaign_threads: role for new channel: %v", err)
	}

	var categoryID string
	settings, settingsErr := models.GetOrCreateGuildSettings(database)
	if settingsErr == nil && messages.IsSnowflake(settings.CampaignsCategoryID) {
		categoryID = settings.CampaignsCategoryID
	} else if settingsErr == nil && messages.IsSnowflake(settings.BillboardCategoryID) {
		categoryID = settings.BillboardCategoryID
	} else {
		var err error
		categoryID, err = findOrCreateCampaignsCategory(s, guildID)
		if err != nil {
			return fmt.Errorf("resolve category: %w", err)
		}
	}
	c.CategoryID = categoryID

	overwrites := []*discordgo.PermissionOverwrite{
		// deny everyone by default
		{
			ID:   guildID,
			Type: discordgo.PermissionOverwriteTypeRole,
			Deny: discordgo.PermissionViewChannel,
		},
		// bot: full control
		{
			ID:    s.State.User.ID,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionManageThreads | discordgo.PermissionSendMessages,
		},
		// DM: ManageThreads so they can post in locked threads
		{
			ID:    c.DungeonMaster,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionManageThreads,
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
	createStandardThreads(database, s, c, ch.ID, channelName)
	return nil
}

/*
SetupExistingChannel links the campaign to an already-existing Discord channel and creates
the standard threads inside it.

Mutates campaign in-place with ChannelID and AnnouncementsThreadID.
*/
func SetupExistingChannel(database *bun.DB, s *discordgo.Session, c *models.Campaign, channelID string) {
	channelName := c.Tag
	if channelName == "" {
		channelName = models.NormalizeTag(c.Name)
	}
	c.ChannelID = channelID
	createStandardThreads(database, s, c, channelID, channelName)
}

// createStandardThreads creates the standard threads in channelID and sets AnnouncementsThreadID.
func createStandardThreads(database *bun.DB, s *discordgo.Session, c *models.Campaign, channelID, channelName string) {
	for _, name := range standardThreads {
		threadName := fmt.Sprintf("%s-%s", channelName, name)
		thread, err := guard.ThreadCreate(s, channelID, threadName, defaultArchiveDuration)
		if err != nil {
			log.Printf("campaign_threads: create thread %s: %v", threadName, err)
			continue
		}
		if name == "announcements" {
			c.AnnouncementsThreadID = thread.ID
		}

		/*
			Lock DM-only threads.
			Players can view but not post. the DM and bot can post because they have ManageThreads on the parent channel.
			This is such that this can be used as a broadcasting channel for new announcements without needing to add a role mention.
		*/
		if name == "welcome" || name == "sessions" || name == "announcements" {
			if err := guard.LockThread(s, thread.ID); err != nil {
				log.Printf("campaign_threads: lock %s thread %s: %v", name, thread.ID, err)
			}
		}

		// Resolve init message: welcome mirrors the billboard body; others use the static map.
		var initMsg string
		if name == "welcome" {
			_, body, _ := helpers.NewCampaignForumPost(database, s, c)
			if body == "" {
				body = fmt.Sprintf(messages.ThreadInitMsgWelcomeFmt, c.Name)
			}
			reminder := "\n" + messages.WelcomeThreadCoverReminder
			combined := body + reminder
			if len([]rune(combined)) > 2000 {
				// Truncate body to fit; keep the reminder intact.
				max := 2000 - len([]rune(reminder)) - 1
				runes := []rune(body)
				if len(runes) > max {
					body = string(runes[:max]) + "…"
				}
				combined = body + reminder
			}
			initMsg = combined
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
	if !messages.IsSnowflake(channelID) {
		log.Printf("campaign_threads: RetireChannel skipped: channelID %q is not a valid snowflake", channelID)
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

	if err := guard.ChannelPermissionSet(s, channelID, guildID,
		discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionViewChannel); err != nil {
		log.Printf("campaign_threads: retire channel %s: deny everyone: %v", channelID, err)
	}

	if campaign.RoleID != "" {
		if err := guard.ChannelPermissionSet(s, channelID, campaign.RoleID,
			discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionViewChannel); err != nil {
			log.Printf("campaign_threads: retire channel %s: deny campaign role: %v", channelID, err)
		}
	}

	for _, roleID := range staffRoleIDs {
		if err := guard.ChannelPermissionSet(s, channelID, roleID,
			discordgo.PermissionOverwriteTypeRole, discordgo.PermissionViewChannel, 0); err != nil {
			log.Printf("campaign_threads: retire channel %s: allow staff role %s: %v", channelID, roleID, err)
		}
	}
}

/*
MoveToArchivedCategory moves a campaign's Discord channel into the admin-configured archived
category.

No-op if ArchivedCategoryID is not configured or the campaign has no channel.

Errors are logged but non-fatal.
*/
func MoveToArchivedCategory(database *bun.DB, s *discordgo.Session, campaign *models.Campaign) {
	if !messages.IsSnowflake(campaign.ChannelID) {
		return
	}
	settings, err := models.GetOrCreateGuildSettings(database)
	if err != nil || !messages.IsSnowflake(settings.ArchivedCategoryID) {
		return
	}
	tag := campaign.Tag
	if tag == "" {
		tag = models.NormalizeTag(campaign.Name)
	}
	archivedName := "archive-" + tag
	if _, err := s.ChannelEditComplex(campaign.ChannelID, &discordgo.ChannelEdit{
		Name:     archivedName,
		ParentID: settings.ArchivedCategoryID,
	}); err != nil {
		log.Printf("campaign_threads: move channel %s to archived category: %v", campaign.ChannelID, err)
	}
}

/*
DeleteBillboard deletes the campaign's billboard forum thread from Discord.
No-op if BillboardThreadID is empty.

Before deleting, it fetches the channel and verifies:
 1. It is a public thread (not a text channel, category, or anything else).
 2. Its parent matches campaign.BillboardChannelID (if that field is set).

This prevents an accidental delete if the stored ID is stale or wrong.

Errors are logged but non-fatal.
*/
func DeleteBillboard(s *discordgo.Session, campaign *models.Campaign) {
	if !messages.IsSnowflake(campaign.BillboardThreadID) {
		return
	}
	ch, err := s.Channel(campaign.BillboardThreadID)
	if err != nil {
		log.Printf("campaign_threads: DeleteBillboard: fetch channel %s: %v", campaign.BillboardThreadID, err)
		return
	}
	if ch.Type != discordgo.ChannelTypeGuildPublicThread && ch.Type != discordgo.ChannelTypeGuildPrivateThread {
		log.Printf("campaign_threads: DeleteBillboard: %s is not a thread (type %d), skipping", campaign.BillboardThreadID, ch.Type)
		return
	}
	if messages.IsSnowflake(campaign.BillboardChannelID) && ch.ParentID != campaign.BillboardChannelID {
		log.Printf("campaign_threads: DeleteBillboard: thread %s parent %s != expected %s, skipping",
			campaign.BillboardThreadID, ch.ParentID, campaign.BillboardChannelID)
		return
	}
	if _, err := s.ChannelDelete(campaign.BillboardThreadID); err != nil {
		log.Printf("campaign_threads: DeleteBillboard: delete thread %s: %v", campaign.BillboardThreadID, err)
	}
}

// billboardChannelName returns the per-format forum channel name for Campaign c.
func billboardChannelName(c *models.Campaign) string {
	if c.IsOneshot {
		return messages.BillboardChannelOneshot
	}
	if c.IsWestmarch {
		return messages.BillboardChannelWestmarch
	}
	return messages.BillboardChannelCampaign
}

/*
findOrCreateForumChannel finds a forum (ChannelTypeGuildForum) channel with the given name inside
categoryID, or creates one if absent.

Follows the same pattern as findOrCreateCampaignsCategory.
*/
func findOrCreateForumChannel(s *discordgo.Session, guildID, categoryID, name string) (string, error) {
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
findForumChannel looks for a forum (ChannelTypeGuildForum) channel with the given name inside
categoryID.

Returns ("", false) if not found. Does not create one.
*/
func findForumChannel(s *discordgo.Session, guildID, categoryID, name string) (string, bool) {
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

/*
PostBillboard resolves the correct billboard forum channel for c's format and creates a forum
thread with the formatted campaign post.

Resolution order:
 1. GuildSettings.BillboardChannel{Format}: explicitly configured by an admin.
 2. findOrCreateForumChannel: find by name inside the Campaigns category, or create.

Callers must db.Update(c) after this returns to persist BillboardChannelID and BillboardThreadID.
*/
func PostBillboard(database *bun.DB, s *discordgo.Session, c *models.Campaign, guildID string) error {
	settings, _ := models.GetOrCreateGuildSettings(database)

	if settings != nil {
		if ch := billboardChannelFromSettings(settings, c); ch != "" {
			return PostBillboardToChannel(database, s, c, ch)
		}
	}

	var categoryID string
	if settings != nil && messages.IsSnowflake(settings.BillboardCategoryID) {
		categoryID = settings.BillboardCategoryID
	} else {
		var err error
		categoryID, err = findOrCreateCampaignsCategory(s, guildID)
		if err != nil {
			return fmt.Errorf("resolve campaigns category: %w", err)
		}
	}

	channelID, err := findOrCreateForumChannel(s, guildID, categoryID, billboardChannelName(c))
	if err != nil {
		return fmt.Errorf("resolve billboard channel: %w", err)
	}

	return PostBillboardToChannel(database, s, c, channelID)
}

// billboardChannelFromSettings returns the configured forum channel ID for c's format, or "".
func billboardChannelFromSettings(s *models.GuildSettings, c *models.Campaign) string {
	if c.IsOneshot {
		return s.BillboardChannelOneshot
	}
	if c.IsWestmarch {
		return s.BillboardChannelWestmarch
	}
	return s.BillboardChannelCampaign
}

/*
PostBillboardToChannel creates a forum thread in an explicitly specified channel and stores
the resulting IDs on c. Used by the import flow when the admin has picked a specific channel.

Callers must db.Update(c) after this returns to persist BillboardChannelID and BillboardThreadID.
*/
func PostBillboardToChannel(database *bun.DB, s *discordgo.Session, c *models.Campaign, channelID string) error {
	title, body, coverURL := helpers.NewCampaignForumPost(database, s, c)
	if title == "" {
		return nil
	}

	threadData := &discordgo.ThreadStart{Name: title, AutoArchiveDuration: defaultArchiveDuration}
	msgData := &discordgo.MessageSend{Content: body}
	if coverURL != "" {
		msgData.Embeds = []*discordgo.MessageEmbed{{Image: &discordgo.MessageEmbedImage{URL: coverURL}}}
	}
	thread, err := s.ForumThreadStartComplex(channelID, threadData, msgData)
	if err != nil {
		return fmt.Errorf("create forum thread: %w", err)
	}

	c.BillboardChannelID = channelID
	c.BillboardThreadID = thread.ID

	if err := db.Update(database, c); err != nil {
		log.Printf("campaign_threads: save billboard IDs for %s: %v", c.ID, err)
	}

	postJoinButton(s, thread.ID, c)

	if messages.IsSnowflake(c.ChannelID) {
		pinBillboardInChannel(s, c.ChannelID, thread.ID)
	}
	return nil
}

/*
postJoinButton sends a standalone pinned message containing only the Join button
to the forum thread. Kept separate from the starter message so clicks don't
overwrite the campaign body.
*/
func postJoinButton(s *discordgo.Session, threadID string, c *models.Campaign) {
	joinComponents := helpers.BillboardComponents(c)
	if len(joinComponents) == 0 {
		return
	}
	_, err := guard.ChannelMessageSendComplex(s, threadID, &discordgo.MessageSend{
		Components: joinComponents,
		Flags:      discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("campaign_threads: send join button to thread %s: %v", threadID, err)
		return
	}
	// if err := guard.ChannelMessagePin(s, threadID, msg.ID); err != nil {
	// 	log.Printf("campaign_threads: pin join button in thread %s: %v", threadID, err)
	// }
}

/*
pinBillboardInChannel sends a message linking to the billboard thread and pins it
in the campaign's private channel so members can navigate to it directly.
*/
func pinBillboardInChannel(s *discordgo.Session, campaignChannelID, threadID string) {
	msg, err := guard.ChannelMessageSend(s, campaignChannelID, fmt.Sprintf(messages.BillboardPinMessage, threadID))
	if err != nil {
		log.Printf("campaign_threads: send billboard pin message to %s: %v", campaignChannelID, err)
		return
	}
	if err := guard.ChannelMessagePin(s, campaignChannelID, msg.ID); err != nil {
		log.Printf("campaign_threads: pin billboard message in %s: %v", campaignChannelID, err)
	}
}

/*
PostCampaignChannelAnnouncement sends a campaign embed to the guild's configured campaign
channel (GuildSettings.CampaignChannelID) when a campaign is approved.

Nothing happens if the channel is not configured.
*/
func PostCampaignChannelAnnouncement(database *bun.DB, s *discordgo.Session, c *models.Campaign, callerID string) {
	settings, err := models.GetOrCreateGuildSettings(database)
	if err != nil || !messages.IsSnowflake(settings.CampaignChannelID) {
		return
	}

	players, err := models.GetCampaignPlayers(database, c.ID)
	if err != nil {
		log.Printf("campaign_threads: load players for announcement of %s: %v", c.ID, err)
	}
	coverURL := models.CoverURLForCampaign(database, c.ID)
	embed := commands.CampaignEmbed(*c, players, coverURL, "", callerID)

	var content string
	if messages.IsSnowflake(c.BillboardThreadID) {
		content = fmt.Sprintf(messages.CampaignAnnouncementThreadFmt, c.BillboardThreadID)
	}

	_, err = guard.ChannelMessageSendComplex(s, settings.CampaignChannelID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("campaign_threads: post announcement for %s to %s: %v", c.ID, settings.CampaignChannelID, err)
	}
}
