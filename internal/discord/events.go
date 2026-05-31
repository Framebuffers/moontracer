package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
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

		players, err := models.GetCampaignPlayers(guildDB, campaign.ID)
		if err != nil {
			log.Printf("events: announcement: load players for %s: %v", campaign.ID, err)
			return
		}

		content := fmt.Sprintf(messages.AnnouncementDMFmt, campaign.Name, m.Author.ID, m.Content)
		for _, p := range players {
			if p.PlayerID == campaign.DungeonMaster {
				continue
			}
			if p.Status != models.StatusActive {
				continue
			}
			d.Push(dispatch.DirectMessage{
				ID:      uuid.NewString(),
				Target:  p.PlayerID,
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

			if err := models.InsertAuditEntry(database, userID, "system", models.AuditCampaignArchive, "DM left server, campaign auto-archived"); err != nil {
				log.Printf("events: failed to write audit entry for campaign %s: %v", cp.CampaignID, err)
			}

			log.Printf("events: auto-archived campaign %s (%s)- DM %s left server", cp.Campaign.Name, cp.CampaignID, userID)
		}
	}
}
