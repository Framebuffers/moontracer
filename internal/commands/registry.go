package commands

import (
	"context"

	"github.com/uptrace/bun"

	"moontracer/internal/dispatch"
	"moontracer/internal/manager/models"
)

/*
All returns every registered command. Add new commands here.

How to add a new slash command:

 1. Create internal/commands/<name>.go with a struct type that holds
    whatever dependencies it needs (usually `db *bun.DB`).

 2. Implement the Command interface (internal/commands/command.go):

    - Data() *discordgo.ApplicationCommand. This is the Discord registration
    metadata (name, description, options).

    - Execute(s, i). This is the handler invoked when the user runs the command.

    - Note: Add user-facing strings to internal/messages/messages.go. Never
    inline literals. The messages package is the single source of truth for copy.

 3. If the command uses autocomplete options, also implement the
    AutocompleteCommand interface (Autocomplete(s, i)).

 4. Add the command to the slice below. Place debug-only commands inside.

 5. If the command needs auth, call auth.Authorize(db, userID, scope, id)
    at the top of Execute. Scopes: ScopePlayer, ScopeDM, ScopeMod.

 6. Run `go build ./...` once all edits are done.
    - Note: Always make sure to also run tests (`go test ./...`) to make sure
    important features, like sovereignty and authorization, work after creating new commands.

Note:

	At startup the bot calls All() once to collect command metadata, registers
	them with Discord (scoped to DISCORD_GUILD_ID when set, global otherwise),
	and routes incoming InteractionApplicationCommand events by command name
	via internal/discord/handler.go.

	Debug-only commands are gated on guard.DevMode so production deployments
	don't expose them in Discord's command picker.
*/
func All(db *bun.DB, d *dispatch.Dispatcher, dataDir, mediaBaseURL string) []Command {
	cmds := []Command{
		&awooCommand{db: db},
		&helpCommand{db: *db, d: d},
		&registerCommand{db: db},
		&meCommand{db: db},
		&tokensCommand{db: db},
		&manageCommand{db: db},
		&nextSessionsCommand{db: db},
		&campaignsCommand{db: db},
		&searchCommand{db: db},
		&banCommand{db: db},
		&unbanCommand{db: db},
		&adminCommand{db: db},
		&aboutCommand{db: db},
		&waosCommand{db: db},
		&newCampaignCommand{db: db},
		&newSessionCommand{db: db},
		&campaignUploadCommand{db: db, dataDir: dataDir, mediaBaseURL: mediaBaseURL},
		&uploadTokenCommand{db: db, dataDir: dataDir, mediaBaseURL: mediaBaseURL},
	}

	return cmds
}

// RegisterCommands populates the commands table with metadata from all registered commands.
func RegisterCommands(db *bun.DB, d *dispatch.Dispatcher) error {
	ctx := context.Background()
	commands := All(db, d, "", "")

	for _, cmd := range commands {
		data := cmd.Data()
		record := &models.CommandRecord{
			Name:        data.Name,
			Description: data.Description,
		}
		_, err := db.NewInsert().Model(record).On("CONFLICT (name) DO NOTHING").Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
