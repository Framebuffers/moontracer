package commands

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
Flow:
 1. Player types /tokens.
 2. Bot queries their Media rows (KindTokenPlayer), most-recent-first, up to 10.
 3. If none exist: ephemeral empty-state message with a Home button.
 4. Otherwise: ephemeral response with one embed per token (title + image) and a
    select menu (token_gallery_select) below for managing individual tokens.
 5. All subsequent select / button interactions are handled by the existing
    token gallery handlers in interactions/player_tokens.go.
*/

/*
tokensCommand opens the player's token gallery directly via slash command.

Mirrors the gallery rendered by the /me hub -> Tokens button (interactions/player_tokens.go
RenderMeTokens), but uses InteractionResponseChannelMessageWithSource instead of
InteractionResponseUpdateMessage so it works as a standalone entry point.
*/
type tokensCommand struct {
	db *bun.DB
}

func (c *tokensCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.TokensCommandName,
		Description: messages.TokensCommandDesc,
	}
}

func (c *tokensCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	var tokens []*models.Media
	if err := c.db.NewSelect().Model(&tokens).
		Where("owner_id = ? AND kind = ?", userID, models.KindTokenPlayer).
		OrderExpr("created_at DESC").
		Limit(10).
		Scan(context.Background()); err != nil {
		log.Printf("tokens: load for %s: %v", userID, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	if len(tokens) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: messages.TokenGalleryNone,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
					}},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var embeds []*discordgo.MessageEmbed
	var options []discordgo.SelectMenuOption
	for _, t := range tokens {
		name := t.Name
		if name == "" {
			name = t.ID[:8]
		}
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title: name,
			Color: messages.EmbedColor,
			Image: &discordgo.MessageEmbedImage{URL: t.URL},
		})
		options = append(options, discordgo.SelectMenuOption{
			Label: name,
			Value: t.ID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messages.TokenGalleryHeader,
			Embeds:  embeds,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    messages.TokenGallerySelectPrefix,
						Placeholder: messages.TokenGallerySelectPlaceholder,
						Options:     options,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
