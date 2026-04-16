package commands

import (
	"context"
	"fmt"
	"log"

	"moontracer/internal/manager/models"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

type waosCommand struct {
	db *bun.DB
}

func (c *waosCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "waos",
		Description: "waos",
	}
}

func (c *waosCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	var record models.CommandRecord

	err := c.db.NewSelect().Model(&record).
		Column("times_used").
		Where("name = ?", "waos").
		Scan(ctx)

	if err != nil {
		log.Printf("waos: smoke test failed: %v", err)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("waosn't. (current counter: %d)", record.TimesUsed),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("waos (x %d)", record.TimesUsed),
		},
	})
}
