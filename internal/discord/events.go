package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// HandleGuildMemberRemove returns a handler that archives any campaigns
// owned by a departing DM. This enforces DM sovereignty: if the DM leaves,
// the campaign becomes an immutable record rather than being handed off.
func HandleGuildMemberRemove(database *bun.DB) func(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	return func(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
		userID := e.User.ID

		// Check if this player is registered.
		_, err := db.GetByID[models.Player](database, userID)
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

			if err := models.InsertAuditEntry(database, userID, "system", models.AuditCampaignArchive, "DM left server, campaign auto-archived"); err != nil {
				log.Printf("events: failed to write audit entry for campaign %s: %v", cp.CampaignID, err)
			}

			log.Printf("events: auto-archived campaign %s (%s) — DM %s left server", cp.Campaign.Name, cp.CampaignID, userID)
		}
	}
}
