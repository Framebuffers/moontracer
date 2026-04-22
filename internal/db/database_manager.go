package db

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type DatabaseManager struct {
	Database *bun.DB
}

func (d *DatabaseManager) Get(connectionString string) (*bun.DB, error) {
	/*
		Note:
		This assumes connectionString is a bare path (no existing query params).
		If the DSN ever includes a "?" (e.g. "file:data/moontracer.db?cache=shared"),
		this concatenation will break — use "&" instead of "?" in that case.

		Right now, the bot passes a bare string, so this is not important *for now*.
	*/
	sqldb, err := sql.Open(sqliteshim.ShimName, connectionString+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())

	err = bunDB.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	d.Database = bunDB
	return bunDB, nil
}
