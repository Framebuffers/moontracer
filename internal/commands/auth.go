package commands

import (
	"moontracer/internal/db"
	"moontracer/internal/manager/models"

	"github.com/uptrace/bun"
)

func isDungeonMaster(database *bun.DB, userID string, campaignID string) (bool, error) {
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		return false, err
	}
	return campaign.DungeonMaster == userID, nil
}

func isRegistered(database *bun.DB, userID string) (bool, error) {
	_, err := db.GetByID[models.Player](database, userID)
	if err != nil {
		return false, nil
	}
	return true, nil
}
