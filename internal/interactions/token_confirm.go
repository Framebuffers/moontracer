package interactions

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/mediaserver"
	"moontracer/internal/messages"
)

/*
tokenApplyHandler opens a naming modal before saving the token.

CustomID: token_apply:{guildID}:{playerID}:{token-uuid}
*/
type tokenApplyHandler struct {
	db           *bun.DB
	dataDir      string
	mediaBaseURL string
}

func (h *tokenApplyHandler) CustomIDPrefix() string {
	return messages.TokenApplyPrefix
}

func (h *tokenApplyHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 4)
	if !ok {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s:%s:%s", messages.TokenApplyModalPrefix, parts[1], parts[2], parts[3]),
			Title:    messages.TokenNameModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.TokenNameFieldID,
						Label:       messages.TokenNameFieldLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.TokenNameFieldPlaceholder,
						Required:    true,
						MaxLength:   100,
					},
				}},
			},
		},
	})
}

/*
tokenApplyModal saves the processed token as a permanent Media record using the
player-provided character name, then removes the src and frm temp files.

CustomID: token_apply_modal:{guildID}:{playerID}:{token-uuid}
*/
type tokenApplyModal struct {
	db           *bun.DB
	dataDir      string
	mediaBaseURL string
}

func (h *tokenApplyModal) CustomIDPrefix() string {
	return messages.TokenApplyModalPrefix
}

func (h *tokenApplyModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 4)
	if !ok {
		return
	}
	guildID, playerID, token := parts[1], parts[2], parts[3]

	var name string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			if ti, ok := comp.(*discordgo.TextInput); ok && ti.CustomID == messages.TokenNameFieldID {
				name = strings.TrimSpace(ti.Value)
			}
		}
	}
	if name == "" {
		name = "token"
	}

	outDisk, outURL := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, "out_"+token, ".png")

	media := &models.Media{
		ID:        uuid.NewString(),
		OwnerID:   playerID,
		Path:      outDisk,
		URL:       outURL,
		Kind:      models.KindTokenPlayer,
		Name:      name,
		MimeType:  "image/png",
		CreatedAt: time.Now(),
	}
	if _, err := h.db.NewInsert().Model(media).Exec(context.Background()); err != nil {
		log.Printf("token_apply_modal: insert failed for player %s: %v", playerID, err)
		helpers.RespondUpdateTerminal(s, i, messages.TokenApplyFailed)
		return
	}

	for _, suffix := range []string{"src_", "frm_"} {
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
			disk, _ := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, suffix+token, ext)
			os.Remove(disk)
		}
	}

	log.Printf("token_apply_modal: token %q saved for player %s, media %s", name, playerID, media.ID)
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.TokenApplySuccess, name))
}

/*
tokenDiscardHandler removes all three temp files and dismisses the preview.

CustomID: token_discard:{guildID}:{playerID}:{token-uuid}
*/
type tokenDiscardHandler struct {
	dataDir      string
	mediaBaseURL string
}

func (h *tokenDiscardHandler) CustomIDPrefix() string {
	return messages.TokenDiscardPrefix
}

func (h *tokenDiscardHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 4)
	if !ok {
		return
	}
	guildID, playerID, token := parts[1], parts[2], parts[3]

	for _, suffix := range []string{"src_", "frm_", "out_"} {
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
			disk, _ := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, suffix+token, ext)
			os.Remove(disk)
		}
	}

	helpers.RespondUpdateTerminal(s, i, messages.TokenDiscardSuccess)
}
