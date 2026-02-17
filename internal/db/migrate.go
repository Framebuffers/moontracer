package db

import (
	"context"
	"log"

	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

// Migrate creates all tables if they don't already exist.
func Migrate(db *bun.DB) error {
	ctx := context.Background()

	tables := []interface{}{
		(*models.Player)(nil),
		(*models.Token)(nil),
		(*models.Campaign)(nil),
		(*models.CampaignPlayer)(nil),
	}

	for _, model := range tables {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}
	}

	log.Println("database migration complete")
	return nil
}
