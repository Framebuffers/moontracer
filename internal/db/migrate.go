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
		(*models.PlayerSettings)(nil),
	}

	for _, model := range tables {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}
	}

	/*
		Migration process (three passes, all idempotent):

		1. CreateTable loop above — `IfNotExists`, so fresh DBs get every column
		   from the model struct tags. Existing DBs skip this entirely.
		2. `alterStmts` below — `ADD COLUMN` retrofits for columns that were added
		   to a model after deploy. Each statement is wrapped by `isDuplicateColumn`:
		   SQLite errors "duplicate column name ..." on re-run, which we swallow.
		   Any other error aborts migration. Columns MUST be appended (never
		   reordered or removed) so old deploys advancing N versions at once
		   replay history in order.
		3. Post-migration cleanup — dedup passes, unique indexes, backfills.
		   Each runs on every boot; must be idempotent.

		Adding a new model:
			- Define it under `internal/manager/models/` with bun tags.
			- Append `(*models.Foo)(nil)` to `tables` (pass 1 handles fresh DBs).
			- If the model will ever gain columns after first deploy, every such
			  addition lands as a new `ALTER TABLE foos ADD COLUMN ...` line at
			  the end of `alterStmts` (pass 2 retrofits existing DBs).
			- If columns need a unique index, backfill, or dedup, add to pass 3.

		Why this shape: bun has no migration framework wired up here, so the
		three passes stand in. Ordering matters within `alterStmts` only when a
		later statement depends on an earlier column existing — otherwise append
		is safe.
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
		"ALTER TABLE campaigns ADD COLUMN cover_channel_id TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN cover_message_id TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN cover_attachment_id TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN cover_cached_url TEXT DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN cover_cached_refreshed TIMESTAMP",
		"ALTER TABLE campaigns ADD COLUMN session_capacity INTEGER NOT NULL DEFAULT 6",
		"UPDATE campaigns SET slots = 2147483647 WHERE is_westmarch = 1 AND slots = -1",
		"ALTER TABLE player_settings ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC'",
		"ALTER TABLE campaign_players ADD COLUMN rsvp_status TEXT NOT NULL DEFAULT ''",
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

	/*
		Refactoring note: deduping commands table.
			Earlier builds (<0.6.2) used ON CONFLICT DO NOTHING with no unique index on (name),
			so every startup inserted fresh rows.
			This coalesces each name's times_used into the min-id row, drops the rest, then adds the
			unique index, so future inserts actually short-circuit.
	*/
	dedupStmts := []string{
		`UPDATE commands SET times_used = (
			SELECT COALESCE(SUM(c2.times_used), 0) FROM commands c2 WHERE c2.name = commands.name
		) WHERE id IN (SELECT MIN(id) FROM commands GROUP BY name)`,
		`DELETE FROM commands WHERE id NOT IN (SELECT MIN(id) FROM commands GROUP BY name)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS commands_name_unique ON commands (name)`,
	}
	for _, stmt := range dedupStmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	log.Println("database migration complete")
	return nil
}

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
