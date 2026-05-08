package interactions

import (
	"context"
	"fmt"
	"log"
	"os"
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
tokenApplyHandler saves the processed token as a permanent Media record and
removes all three temp files (source, frame, out).

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
	guildID, playerID, token := parts[1], parts[2], parts[3]

	outDisk, outURL := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, "out_"+token, ".png")

	media := &models.Media{
		ID:        uuid.NewString(),
		OwnerID:   playerID,
		Path:      outDisk,
		URL:       outURL,
		Kind:      models.KindTokenPlayer,
		Name:      "token.png",
		MimeType:  "image/png",
		CreatedAt: time.Now(),
	}
	if _, err := h.db.NewInsert().Model(media).Exec(context.Background()); err != nil {
		log.Printf("token_apply: insert failed for player %s: %v", playerID, err)
		helpers.RespondUpdateTerminal(s, i, messages.TokenApplyFailed)
		return
	}

	for _, suffix := range []string{"src_", "frm_"} {
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
			disk, _ := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, suffix+token, ext)
			os.Remove(disk)
		}
	}

	log.Printf("token_apply: token saved for player %s, media %s", playerID, media.ID)
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.TokenApplySuccess))
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
