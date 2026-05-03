package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/scheduler"
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

			log.Printf("events: auto-archived campaign %s (%s) — DM %s left server", cp.Campaign.Name, cp.CampaignID, userID)
		}
	}
}
