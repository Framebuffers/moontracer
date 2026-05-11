package interactions

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
RenderMeTokens renders the token gallery: up to 10 embeds (one per token) with a manage select below.

Navigated to via ViewMeTokens. Back goes to ViewMe.

Note:

	Discord limits files/embeds being shown inside ephemeral messages to 10.
*/
func RenderMeTokens(s *discordgo.Session, i *discordgo.InteractionCreate, bdb *bun.DB, userID string) {
	var tokens []*models.Media
	if err := bdb.NewSelect().Model(&tokens).
		Where("owner_id = ? AND kind = ?", userID, models.KindTokenPlayer).
		OrderExpr("created_at DESC").
		Limit(10).
		Scan(context.Background()); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if len(tokens) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    messages.TokenGalleryNone,
				Embeds:     []*discordgo.MessageEmbed{},
				Components: []discordgo.MessageComponent{helpers.BackRow(router.ViewMe)},
				Flags:      discordgo.MessageFlagsEphemeral,
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
		Type: discordgo.InteractionResponseUpdateMessage,
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
				helpers.BackRow(router.ViewMe),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
tokenGallerySelectHandler shows a detail view for the selected token with Assign and Delete options.

CustomID: token_gallery_select
*/
type tokenGallerySelectHandler struct {
	db *bun.DB
}

func (h *tokenGallerySelectHandler) CustomIDPrefix() string { return messages.TokenGallerySelectPrefix }

func (h *tokenGallerySelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	mediaID := values[0]

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	name := media.Name
	if name == "" {
		name = media.ID[:8]
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title: name,
				Color: messages.EmbedColor,
				Image: &discordgo.MessageEmbedImage{URL: media.URL},
			}},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.PlayerTokenAssignLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s", messages.TokenGalleryAssignPrefix, mediaID),
					},
					discordgo.Button{
						Label:    messages.TokenDownloadLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("%s:%s", messages.TokenDownloadPrefix, mediaID),
					},
					discordgo.Button{
						Label:    messages.TokenDeleteLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.TokenDeletePromptPrefix, mediaID),
					},
				}},
				helpers.BackRow(router.ViewMeTokens),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
tokenGalleryAssignHandler shows a campaign picker to assign the chosen token to.

CustomID: token_gallery_assign:<mediaID>
*/
type tokenGalleryAssignHandler struct {
	db *bun.DB
}

func (h *tokenGalleryAssignHandler) CustomIDPrefix() string { return messages.TokenGalleryAssignPrefix }

func (h *tokenGalleryAssignHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	mediaID := parts[1]
	userID := helpers.GetUserID(i)

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	campaigns, err := models.GetPlayerCampaigns(h.db, userID)
	if err != nil || len(campaigns) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.NoCampaignsMessage)
		return
	}

	name := media.Name
	if name == "" {
		name = media.ID[:8]
	}

	var options []discordgo.SelectMenuOption
	for _, cp := range campaigns {
		if cp.Campaign == nil {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: cp.Campaign.Name,
			Value: cp.CampaignID,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.TokenGalleryAssignPrompt, name),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    fmt.Sprintf("%s:%s", messages.TokenGalleryAssignSelectPrefix, mediaID),
						Placeholder: messages.TokenGalleryAssignSelectPlaceholder,
						Options:     options,
					},
				}},
				helpers.BackRow(router.ViewMeTokens),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
tokenGalleryAssignSelectHandler executes the token-to-campaign assignment from the gallery.

CustomID: token_gallery_assign_select:<mediaID>
*/
type tokenGalleryAssignSelectHandler struct {
	db *bun.DB
}

func (h *tokenGalleryAssignSelectHandler) CustomIDPrefix() string {
	return messages.TokenGalleryAssignSelectPrefix
}

func (h *tokenGalleryAssignSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	mediaID := parts[1]
	userID := helpers.GetUserID(i)

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	campaignID := values[0]

	if _, err := h.db.NewUpdate().Model((*models.CampaignPlayer)(nil)).
		Set("media_id = ?", mediaID).
		Where("player_id = ? AND campaign_id = ?", userID, campaignID).
		Exec(context.Background()); err != nil {
		log.Printf("token_gallery_assign_select: update player %s campaign %s: %v", userID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.PlayerTokenAssignSuccess)
		return
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.PlayerTokenAssignSuccess, campaign.Name))
}

/*
tokenDeletePromptHandler shows a delete confirmation for the selected token.

CustomID: token_delete_prompt:<mediaID>
*/
type tokenDeletePromptHandler struct {
	db *bun.DB
}

func (h *tokenDeletePromptHandler) CustomIDPrefix() string { return messages.TokenDeletePromptPrefix }

func (h *tokenDeletePromptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	mediaID := parts[1]

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	name := media.Name
	if name == "" {
		name = media.ID[:8]
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.TokenDeleteConfirmMsg, name),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.TokenDeleteConfirmLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.TokenDeleteConfirmPrefix, mediaID),
					},
					router.NavButton(messages.TokenDeleteCancelLabel, discordgo.SecondaryButton, router.ViewMeTokens),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
tokenDeleteConfirmHandler deletes the token record and its file, clearing any campaign assignments.
It clear tokens from all campaign slots before deleting the record.

CustomID: token_delete_confirm:<mediaID>
*/
type tokenDeleteConfirmHandler struct {
	db      *bun.DB
	dataDir string
}

func (h *tokenDeleteConfirmHandler) CustomIDPrefix() string { return messages.TokenDeleteConfirmPrefix }

func (h *tokenDeleteConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	mediaID := parts[1]

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	name := media.Name
	if name == "" {
		name = media.ID[:8]
	}

	if _, err := h.db.NewUpdate().Model((*models.CampaignPlayer)(nil)).
		Set("media_id = ''").
		Where("media_id = ?", mediaID).
		Exec(context.Background()); err != nil {
		log.Printf("token_delete_confirm: clear campaign slots for media %s: %v", mediaID, err)
	}

	if _, err := h.db.NewDelete().Model((*models.Media)(nil)).
		Where("id = ?", mediaID).
		Exec(context.Background()); err != nil {
		log.Printf("token_delete_confirm: delete media record %s: %v", mediaID, err)
		helpers.RespondUpdateTerminal(s, i, messages.TokenDeleteFailed)
		return
	}

	if err := os.Remove(media.Path); err != nil && !os.IsNotExist(err) {
		log.Printf("token_delete_confirm: remove file %s: %v", media.Path, err)
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.TokenDeleteSuccess, name))
}

/*
tokenDownloadHandler sends a single token file as an ephemeral attachment.

CustomID: token_download:<mediaID>
*/
type tokenDownloadHandler struct {
	db *bun.DB
}

func (h *tokenDownloadHandler) CustomIDPrefix() string { return messages.TokenDownloadPrefix }

func (h *tokenDownloadHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	mediaID := parts[1]

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	f, err := os.Open(media.Path)
	if err != nil {
		log.Printf("token_download: open %s: %v", media.Path, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	defer f.Close()

	name := media.Name
	if name == "" {
		name = media.ID[:8]
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.PlayerDownloadContent, 1),
			Embeds:  []*discordgo.MessageEmbed{},
			Files: []*discordgo.File{{
				Name:        name + ".png",
				ContentType: "image/png",
				Reader:      f,
			}},
			Components: []discordgo.MessageComponent{
				helpers.BackRow(router.ViewMeTokens),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
