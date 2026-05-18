package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. User clicks `/mycampaigns`, getting a list of which ones the Player is a part of.
		2. User clicks a button, like `campaign_view:suvachi`
		3. `campaignView` catches it, parses the tag given by `CustomID`
		4. Fetches Player and Campaign data from the DB.
		5. Respons with an ephemeral view (only the player can see it).

*/

/*
campaignView is a ComponentHandler that shows campaign details **only to the user** through an ephemeral embed.

It's a quick preview of the campaigns a user is present on.

The difference with `/campaign` is that it is a quick, private, simplified view of a Campaign data.
*/
type campaignView struct {
	db *bun.DB
}

// CustomIDPrefix is the unique identifier used to route it through `handler.go`, identifying it as a ComponentHandler.
func (h *campaignView) CustomIDPrefix() string {
	return "campaign_view"
}

// HandleComponents handles the process of fetching data from the DB and composing the final interaction to be returned to Discord.
func (h *campaignView) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	tag := parts[1]

	campaign, err := db.GetByTag[models.Campaign](h.db, tag)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if !campaign.IsApproved {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaign.ID)
	if err != nil {
		log.Printf("campaign_view: %s: %v", messages.PlayerFetchErrorMessage, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignLoadFailureErrorMessage)
		return
	}

	coverURL := models.CoverURLForCampaign(h.db, campaign.ID)
	embed := buildCampaignEmbed(*campaign, players, coverURL)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "",
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{helpers.BackRow(router.ViewMyCampaigns)},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// buildCampaignEmbed builds the ephemeral embed that the user will see, with all the Campaigns the Player is a player on.
func buildCampaignEmbed(c models.Campaign, players []models.CampaignPlayer, coverURL string) *discordgo.MessageEmbed {
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
		playerLines = append(playerLines, fmt.Sprintf("<@%s> - %s (%s)", p.PlayerID, p.Role, p.Status))
	}
	playersValue := "None"
	if len(playerLines) > 0 {
		playersValue = strings.Join(playerLines, "\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - %s", campaignType, c.Name),
		Description: c.Description,
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Tag", Value: c.Tag, Inline: true},
			{Name: "DM", Value: fmt.Sprintf("<@%s>", c.DungeonMaster), Inline: true},
			{Name: "Status", Value: status, Inline: true},
			{Name: "Slots", Value: c.DisplaySlots(), Inline: true},
			{Name: "Edition", Value: c.Game.Edition, Inline: true},
			{Name: fmt.Sprintf("Players (%d)", len(players)), Value: playersValue, Inline: false},
		},
	}
	if coverURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: coverURL}
	}
	return embed
}
