package db

import (
	"context"
	"log"
	"strings"

	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/manager/models"
)

// Migrate creates all tables if they don't already exist.
func Migrate(db *bun.DB) error {
	ctx := context.Background()

	tables := []interface{}{
		(*models.CommandRecord)(nil),
		(*models.Player)(nil),
		(*models.Media)(nil),
		(*models.Campaign)(nil),
		(*models.CampaignPlayer)(nil),
		(*models.AuditEntry)(nil),
		(*models.PlayerSettings)(nil),
		(*models.Session)(nil),
		(*models.SessionRSVP)(nil),
		(*models.GuildSettings)(nil),
	}

	for _, model := range tables {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}
	}

	/*
		Migration process (three passes, all idempotent):

		1. CreateTable loop above -`IfNotExists`, so fresh DBs get every column
		   from the model struct tags. Existing DBs skip this entirely.
		2. `alterStmts` below -`ADD COLUMN` retrofits for columns that were added
		   to a model after deploy. Each statement is wrapped by `isDuplicateColumn`:
		   SQLite errors "duplicate column name ..." on re-run, which we swallow.
		   Any other error aborts migration. Columns MUST be appended (never
		   reordered or removed) so old deploys advancing N versions at once
		   replay history in order.
		3. Post-migration cleanup -dedup passes, unique indexes, backfills.
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
		later statement depends on an earlier column existing -otherwise append
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
		"ALTER TABLE campaign_players ADD COLUMN media_id TEXT",
		"ALTER TABLE media ADD COLUMN url TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN player_sheet_url TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaign_players ADD COLUMN sheet_url TEXT",
		"ALTER TABLE campaigns ADD COLUMN deleted_at TIMESTAMP",
		"ALTER TABLE campaigns ADD COLUMN billboard_channel_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE campaigns ADD COLUMN billboard_thread_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE guild_settings ADD COLUMN billboard_category_id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE guild_settings ADD COLUMN campaign_channel_id TEXT NOT NULL DEFAULT ''",
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

	// Unique index so the conflict-based seed below stays idempotent.
	if _, err := db.ExecContext(ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS sessions_campaign_scheduled_unique ON sessions (campaign_id, scheduled_at)`,
	); err != nil {
		return err
	}

	/*
		Backfill one session row per approved, non-archived campaign that already has a
		NextSession set and has no sessions in the new table yet.
	*/
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO sessions (id, campaign_id, scheduled_at, title, channel_msg_id, capacity, alert_sent, status, created_at)
		SELECT lower(hex(randomblob(16))), c.id, c.next_session, '', '', 0, c.alert_sent, 'upcoming', datetime('now')
		FROM campaigns c
		WHERE c.next_session IS NOT NULL
		  AND c.next_session > datetime('now')
		  AND c.is_approved = 1
		  AND c.is_archived = 0
		  AND NOT EXISTS (SELECT 1 FROM sessions s WHERE s.campaign_id = c.id)
	`); err != nil {
		return err
	}

	log.Println("database migration complete")
	return nil
}

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
