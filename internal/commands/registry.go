package commands

import (
	"context"

	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
)

// All returns every registered command. Add new commands here.
func All(db *bun.DB) []Command {
	return []Command{
		&pingCommand{},
		&awooCommand{},
		&campaignCommand{db: db},
		&playerCommand{db: db},
		&registerCommand{db: db},
		&newCampaign{db: db},
	}
}

// RegisterCommands populates the commands table with metadata from all registered commands.
func RegisterCommands(db *bun.DB) error {
	ctx := context.Background()
	commands := All(db)

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
