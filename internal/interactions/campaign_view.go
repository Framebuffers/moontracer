package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type campaignView struct {
	db *bun.DB
}

func (h *campaignView) CustomIDPrefix() string {
	return "campaign_view"
}

func (h *campaignView) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.MessageComponentData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[1]

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondInteraction(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("%s %v", messages.PlayerFetchErrorMessage, err)
		respondInteraction(s, i, messages.CampaignLoadFailureErrorMessage)
		return
	}

	embed := buildCampaignEmbed(*campaign, players)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func buildCampaignEmbed(c models.Campaign, players []models.CampaignPlayer) *discordgo.MessageEmbed {
	status := "Closed"
	if c.IsOpen {
		status = "Open"
	}

	campaignType := "Campaign"
	if c.IsOneshot {
		campaignType = "One-shot"
	}

	var playerLines []string
	for _, p := range players {
		playerLines = append(playerLines, fmt.Sprintf("<@%s> — %s (%s)", p.PlayerID, p.Role, p.Status))
	}
	playersValue := "None"
	if len(playerLines) > 0 {
		playersValue = strings.Join(playerLines, "\n")
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s — %s", campaignType, c.ID),
		Description: c.Description,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "DM", Value: fmt.Sprintf("<@%s>", c.DungeonMaster), Inline: true},
			{Name: "Status", Value: status, Inline: true},
			{Name: "Slots", Value: fmt.Sprintf("%d", c.Slots), Inline: true},
			{Name: "Edition", Value: c.Game.Edition, Inline: true},
			{Name: fmt.Sprintf("Players (%d)", len(players)), Value: playersValue, Inline: false},
		},
	}
}
