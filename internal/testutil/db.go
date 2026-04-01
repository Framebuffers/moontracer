package testutil

import (
	"database/sql"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"moontracer/internal/db"
)

// NewTestDB creates a fresh in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())

	if err := db.Migrate(bunDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Cleanup(func() { bunDB.Close() })

	return bunDB
}
