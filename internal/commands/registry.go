package commands

import (
	"context"

	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/manager/models"
)

/*
All returns every registered command. Add new commands here.

Note:

	Debug-only commands are gated on guard.DevMode so production deployments
	don't expose them in Discord's command picker.
*/
func All(db *bun.DB, d *dispatch.Dispatcher) []Command {
	cmds := []Command{
		&pingCommand{},
		&awooCommand{db: db},
		&helpCommand{db: *db, d: d},
		&playerCommand{db: db},
		&registerCommand{db: db},
		&meCommand{db: db},
		&campaignsCommand{db: db},
		&searchCommand{db: db},
		&newCampaign{db: db},
		&banCommand{db: db},
		&unbanCommand{db: db},
		&manageCampaigns{db: db},
		&adminCommand{db: db},
		&aboutCommand{db: db},
		&waosCommand{db: db},
	}

	if guard.DevMode {
		cmds = append(cmds, &campaignDatabaseCommand{db: db})
	}

	return cmds
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
