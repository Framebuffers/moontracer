package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/manager/models"
)

/*
ArchiveCampaign sets a campaign as archived and marks all members as finished.
Shared between the manage_archive handler and the GuildMemberRemove event handler.
*/
func ArchiveCampaign(database *bun.DB, campaign *models.Campaign, reason string) error {
	campaign.IsArchived = true
	campaign.ArchivedAt = time.Now().UTC()
	campaign.ArchivedReason = reason

	if err := db.Update(database, campaign); err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}

	players, err := models.GetCampaignPlayers(database, campaign.ID)
	if err != nil {
		return fmt.Errorf("get campaign players: %w", err)
	}

	for _, p := range players {
		if p.Status == models.StatusActive || p.Status == models.StatusPending || p.Status == models.StatusHiatus {
			if err := models.SetCampaignPlayerStatus(database, p.PlayerID, campaign.ID, models.StatusFinished); err != nil {
				log.Printf("archive: failed to set player %s to finished in campaign %s: %v", p.PlayerID, campaign.ID, err)
			}
		}
	}

	return nil
}
