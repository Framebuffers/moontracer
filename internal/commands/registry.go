package commands

import (
	"context"

	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	"moontracer/internal/manager/models"
)

// All returns every registered command. Add new commands here.
func All(db *bun.DB, d *dispatch.Dispatcher) []Command {
	return []Command{
		&pingCommand{},
		&awooCommand{db: db},
		&helpCommand{db: *db, d: d},
		&campaignCommand{db: db},
		&playerCommand{db: db},
		&registerCommand{db: db},
		&newCampaign{db: db},
		&addPlayer{db: db},
		&banCommand{db: db},
		&unbanCommand{db: db},
		&manageCampaigns{db: db},
		&setCampaignRole{db: db},
		&abandonCampaign{db: db},
	}
}

// RegisterCommands populates the commands table with metadata from all registered commands.
func RegisterCommands(db *bun.DB, d *dispatch.Dispatcher) error {
	ctx := context.Background()
	commands := All(db, d)

	for _, cmd := range commands {
		data := cmd.Data()
		record := &models.CommandRecord{
			Name:        data.Name,
			Description: data.Description,
		}
		_, err := db.NewInsert().Model(record).On("CONFLICT DO NOTHING").Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
