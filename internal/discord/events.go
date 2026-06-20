package discord

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"github.com/framebuffers/moontracer/internal/auditlog"
	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/cooldown"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/scheduler"
)

/*
	Flow:
		1. A guild member leaves the server, triggering `GuildMemberRemove`.
		2. Resolve the guild's database from the event's guild ID.
		3. Check if the departing user is a registered player. If not, do nothing.
		4. Load all CampaignPlayer entries for this player.
		5. For each entry where the player is the DM and the campaign is still mutable:
			a. Archive the campaign (set IsArchived, ArchivedReason).
			b. Insert an audit entry recording the auto-archive.
		6. Log each archived campaign.
*/

/*
	HandleGuildMemberRemove returns a handler that archives any campaigns owned by a departing DM.

This enforces DM sovereignty: if the DM leaves, the campaign becomes an immutable record rather than being handed off.
*/
/*
HandleAnnouncementMessage sends a DM's post in an announcements thread to all active campaign members via DM.

When the DM posts in the campaign's announcements thread, every active player receives a DM with the message content.

The bot itself and the DM are excluded.
*/
func HandleAnnouncementMessage(guildDBM *db.GuildDBManager, d *dispatch.Dispatcher) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}

		guildDB, err := guildDBM.GetOrCreate(m.GuildID)
		if err != nil {
			return
		}

		campaign, err := models.GetCampaignByAnnouncementsThreadID(guildDB, m.ChannelID)
		if err != nil {
			return
		}

		if m.Author.ID != campaign.DungeonMaster {
			return
		}

		// 5min cooldown
		if !cooldown.Global.Allow("announce:"+campaign.ID, 5*time.Minute) {
			remaining := cooldown.Global.Remaining("announce:" + campaign.ID)
			log.Printf("events: announcement cooldown active for campaign %s (%s remaining)", campaign.ID, cooldown.FormatRemaining(remaining))
			return
		}

		players, err := models.GetCampaignPlayers(guildDB, campaign.ID)
		if err != nil {
			log.Printf("events: announcement: load players for %s: %v", campaign.ID, err)
			return
		}

		content := fmt.Sprintf(messages.AnnouncementDMFmt, campaign.Name, m.Author.ID, m.Content)

		sent := map[string]bool{}
		for _, p := range players {
			if p.PlayerID == campaign.DungeonMaster {
				continue
			}
			if p.Status != models.StatusActive {
				continue
			}
			sent[p.PlayerID] = true
			d.Push(dispatch.DirectMessage{
				ID:      uuid.NewString(),
				Target:  p.PlayerID,
				Content: content,
			})
		}

		/*
			send to the DM when it's on test mode, so they can see if this works
		*/
		if aid := guard.DebugAdminID; aid != "" && !sent[aid] {
			log.Printf("events: announcement debug: testing announcements for campaign %s to DEBUG_ADMIN_ID", campaign.ID)
			d.Push(dispatch.DirectMessage{
				ID:      uuid.NewString(),
				Target:  aid,
				Content: content,
			})
		}
	}
}

func HandleGuildMemberRemove(guildDBM *db.GuildDBManager, sched *scheduler.Scheduler) func(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
		database, err := guildDBM.GetOrCreate(e.GuildID)
		if err != nil {
			log.Printf("events: failed to get DB for guild %s: %v", e.GuildID, err)
			return
		}

		userID := e.User.ID

		// Check if this player is registered.
		_, err = db.GetByID[models.Player](database, userID)
		if err != nil {
			return // not registered, nothing to do
		}

		// Load all campaigns this player participates in.
		campaignPlayers, err := models.GetPlayerCampaigns(database, userID)
		if err != nil {
			log.Printf("events: failed to load campaigns for departing member %s: %v", userID, err)
			return
		}

		for _, cp := range campaignPlayers {
			if cp.Role != models.RoleDM {
				continue
			}
			if cp.Campaign == nil {
				continue
			}
			if !cp.Campaign.CanMutate() {
				continue // already archived
			}

			if err := commands.ArchiveCampaign(database, cp.Campaign, messages.AbandonReasonLeftServer); err != nil {
				log.Printf("events: failed to archive campaign %s for departing DM %s: %v", cp.CampaignID, userID, err)
				continue
			}
			sched.Cancel(e.GuildID, cp.CampaignID)

			auditlog.Post(s, database, e.GuildID, userID, "system", models.AuditCampaignArchive, "DM left server, campaign auto-archived")

			log.Printf("events: auto-archived campaign %s (%s)- DM %s left server", cp.Campaign.Name, cp.CampaignID, userID)
		}
	}
}

// joinTriggers are the strings used when a user wants to join a Campaign by typing a specific word.
var joinTriggers = map[string]bool{"me": true, "yo": true}

/*
HandleBillboardMessage watches every message in a campaign's billboard thread.

If the message content matches a join trigger, the author is automatically registered
(if not already) and joins the campaign.
*/
func HandleBillboardMessage(guildDBM *db.GuildDBManager, d *dispatch.Dispatcher) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}
		if !joinTriggers[strings.ToLower(strings.TrimSpace(m.Content))] {
			return
		}

		guildDB, err := guildDBM.GetOrCreate(m.GuildID)
		if err != nil {
			return
		}

		campaign, err := models.GetCampaignByBillboardThreadID(guildDB, m.ChannelID)
		if err != nil {
			return // not a billboard thread we know about
		}

		userID := m.Author.ID

		// auto-register non-members of the bot
		if _, err := db.GetByID[models.Player](guildDB, userID); err != nil {
			player := &models.Player{ID: userID}
			if err := db.Insert(guildDB, player); err != nil {
				log.Printf("events: billboard join: auto-register %s: %v", userID, err)
				// try joining anyway
			}
		}

		if !campaign.IsApproved || !campaign.CanMutate() {
			return
		}

		players, err := models.GetCampaignPlayers(guildDB, campaign.ID)
		if err != nil {
			log.Printf("events: billboard join: load players for %s: %v", campaign.ID, err)
			return
		}
		for _, p := range players {
			if p.PlayerID == userID {
				return // already in or banned
			}
		}

		if !campaign.IsWestmarch && campaign.Slots > 0 {
			active := 0
			for _, p := range players {
				if p.Status == models.StatusActive {
					active++
				}
			}
			if active >= campaign.Slots {
				d.Push(dispatch.DirectMessage{
					ID:      uuid.NewString(),
					Target:  userID,
					Content: messages.CampaignFullMessage,
				})
				return
			}
		}

		cp := &models.CampaignPlayer{
			PlayerID:   userID,
			CampaignID: campaign.ID,
			Role:       models.RolePlayer,
			Status:     models.StatusActive,
		}
		if err := db.Insert(guildDB, cp); err != nil {
			log.Printf("events: billboard join: insert player %s into %s: %v", userID, campaign.ID, err)
			return
		}

		if campaign.RoleID != "" {
			if err := guard.GuildMemberRoleAdd(s, m.GuildID, userID, campaign.RoleID); err != nil {
				log.Printf("events: billboard join: assign role to %s: %v", userID, err)
			}
		}

		d.Push(dispatch.DirectMessage{
			ID:      uuid.NewString(),
			Target:  userID,
			Content: fmt.Sprintf(messages.PlayerJoinedCampaignMessage, campaign.Name),
		})
	}
}
