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
		(*models.AuditEntry)(nil),
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
		"ALTER TABLE campaigns ADD COLUMN can_overflow INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaigns ADD COLUMN role_id TEXT DEFAULT ''",
		"ALTER TABLE commands ADD COLUMN times_used INTEGER NOT NULL DEFAULT 0",

		// campaign_players columns added after initial schema.
		// Schedule fields for campaigns.
		"ALTER TABLE campaigns ADD COLUMN day_of_week INTEGER NOT NULL DEFAULT -1",
		"ALTER TABLE campaigns ADD COLUMN start_time TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN duration_hours REAL NOT NULL DEFAULT 3",

		// Player timezone (deferred — defaults to UTC for now).
		"ALTER TABLE players ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC'",

		// campaign_players columns added after initial schema.
		"ALTER TABLE campaign_players ADD COLUMN ban_reason TEXT",
		"ALTER TABLE campaign_players ADD COLUMN banned_from_campaign INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaign_players ADD COLUMN ban_reason_from_campaign TEXT",
		"ALTER TABLE campaign_players ADD COLUMN sessions_played INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaign_players ADD COLUMN dice_throw_picture TEXT",
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
