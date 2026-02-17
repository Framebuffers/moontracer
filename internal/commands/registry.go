package commands

import "github.com/uptrace/bun"

// All returns every registered command. Add new commands here.
func All(db *bun.DB) []Command {
	return []Command{
		&pingCommand{},
		&awooCommand{},
		&campaignCommand{db: db},
		&playerCommand{db: db},
	}
}
