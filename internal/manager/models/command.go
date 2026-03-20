package models

import "github.com/uptrace/bun"

// CommandRecord stores metadata about registered slash commands in the database.
type CommandRecord struct {
	bun.BaseModel `bun:"table:commands"`

	ID          int    `bun:",pk,autoincrement"`
	Name        string `bun:",notnull"`
	Description string
	TimesUsed   int    `bun:",notnull,default:0"`
}
