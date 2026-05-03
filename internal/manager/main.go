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

// populateDB opens or creates a DB for persistent data.
// Foreign Key constraint enforcement is enabled. This will make SQLite reject any insert/update operation that has an invalid Campaign ID,
// and to prevent deleting a campaign with players still on it by accident (orphaned records).
func populateDB() {
	sqldb, err := sql.Open(sqliteshim.ShimName, "file:data/moontracer.db?_pragma=foreign_keys(1)")
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()

	db.NewCreateTable().Model((*models.Player)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.Media)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.Campaign)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*models.CampaignPlayer)(nil)).IfNotExists().Exec(ctx)
}
