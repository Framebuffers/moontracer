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

	/*
		Init Procedure:
			Campaigns -> Players -> Commands -> Add scheduling to Campaigns -> CampaignPlayers -> Audit/Archival
	*/

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
		"ALTER TABLE campaigns ADD COLUMN day_of_week INTEGER NOT NULL DEFAULT -1",
		"ALTER TABLE campaigns ADD COLUMN start_time TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN duration_hours REAL NOT NULL DEFAULT 3",
		"ALTER TABLE players ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC'",
		"ALTER TABLE campaign_players ADD COLUMN ban_reason TEXT",
		"ALTER TABLE campaign_players ADD COLUMN banned_from_campaign INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaign_players ADD COLUMN ban_reason_from_campaign TEXT",
		"ALTER TABLE campaign_players ADD COLUMN sessions_played INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaign_players ADD COLUMN dice_throw_picture TEXT",
		"ALTER TABLE campaigns ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE campaigns ADD COLUMN archived_at TIMESTAMP",
		"ALTER TABLE campaigns ADD COLUMN archived_reason TEXT",
		"ALTER TABLE campaigns ADD COLUMN channel_id TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN category_id TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN announcements_thread_id TEXT DEFAULT ''",
	}
	for _, stmt := range alterStmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if !isDuplicateColumn(err) {
				return err
			}
		}
	}

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
