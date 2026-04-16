package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Test command:
		- Checks for connectivity with the VPS/Docker container.
		- Increments and displays an awoo counter (also tests DB read/write).
		- Note: the bot uses WebHooks only.
*/

type awooCommand struct {
	db *bun.DB
}

// Data is the command metadata that Discord shows to users.
func (c *awooCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.AwooCommandName,
		Description: messages.AwooCommandDesc,
	}
}

// Execute is the logic that runs when the user invokes that command.
func (c *awooCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	var record models.CommandRecord

	err := c.db.NewSelect().Model(&record).
		Column("times_used").
		Where("name = ?", "awoo").
		Scan(ctx)

	if err != nil {
		log.Printf("awoo: smoke test failed: %v", err)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("there was no one to awoo to. (current counter: %d)", record.TimesUsed),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("awoo (x %d)", record.TimesUsed),
		},
	})
}
