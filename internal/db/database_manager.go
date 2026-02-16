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
	sqldb, err := sql.Open(sqliteshim.ShimName, connectionString)
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
