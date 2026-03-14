package db

import (
	"context"
	"log"
	"strings"

	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

// Migrate creates all tables if they don't already exist.
func Migrate(db *bun.DB) error {
	ctx := context.Background()

	tables := []interface{}{
		(*models.CommandRecord)(nil),
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

	// Add columns that may be missing from earlier schema versions.
	// This fixes an issue where, when deploying the bot, it didn't migrate new commands or players.
	alterStmts := []string{
		"ALTER TABLE campaigns ADD COLUMN name TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN tag TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN is_approved INTEGER NOT NULL DEFAULT 0",
		"UPDATE campaigns SET is_approved = 1 WHERE is_approved = 0",
		"ALTER TABLE players ADD COLUMN role TEXT NOT NULL DEFAULT 'player'",
		"ALTER TABLE players ADD COLUMN is_banned INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE players ADD COLUMN ban_reason TEXT",
	}
	for _, stmt := range alterStmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if !isDuplicateColumn(err) {
				return err
			}
		}
	}

	// Ensure the unique index on tag exists.
	_, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS campaigns_tag_unique ON campaigns (tag)")
	if err != nil {
		return err
	}

	log.Println("database migration complete")
	return nil
}

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
