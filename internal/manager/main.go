package manager

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"moontracer/internal/manager/models"
)

func main() {
	populateDB()
}

func populateDB() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:data/moontracer.db?_pragma=foreign_keys(1)")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()

	db.NewCreateTable().Model((*models.Player)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.Token)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.Campaign)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.CampaignPlayer)(nil)).IfNotExists().Exec(ctx)
}
