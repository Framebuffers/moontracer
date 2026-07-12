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

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/mediaserver"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		Triggered post-upload token confirmation, started by /uploadtoken.
		1. /uploadtoken processes the source+frame attachments
			a. writes src_/frm_/out_ temp files under dataDir, and
			b. posts a preview embed with Apply/Discard buttons (handled in commands/upload_token.go).
		2. Apply click: tokenApplyHandler opens a naming modal
		   (CustomID: token_apply:<guildID>:<playerID>:<token-uuid>).
		3. Modal submit: tokenApplyModal:
		    a. Inserts a Media row (KindTokenPlayer) pointing at the out_ file.
		    b. Removes the src_/frm_ temps (out_ becomes the canonical file).
		    c. If the player has active non-DM campaigns, renders a campaign
		       select + Skip button so the token can be assigned immediately.
		       Otherwise replies with a terminal "saved" message.
		4. Campaign select: playerTokenPostcreateSelectHandler updates
		   CampaignPlayer.media_id for (player, campaign).
		5. Skip click: playerTokenSkipHandler dismisses with "saved, not assigned".
		6. Discard click (any time before Apply): tokenDiscardHandler removes
		   all three temp files (src_/frm_/out_) and replies with a dismissal.

	Notes:
		- Re-assignment of an existing token is handled in player_tokens.go
		  (gallery flow), not here.
		- File extensions are probed across {.jpg,.jpeg,.png,.webp} because the
		  source upload can be any of these; out_ is always .png.
*/

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
tokenApplyModal saves the processed token, then offers to assign it to a campaign.

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

	if helpers.GetUserID(i) != playerID {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

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

	// Load campaigns where the player is an active non-DM member.
	homeRow := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
		}},
	}

	allCPs, err := models.GetPlayerCampaigns(h.db, playerID)
	if err != nil {
		respondWithTokenFile(s, i, outDisk, name, fmt.Sprintf(messages.TokenApplySuccess, name), homeRow)
		return
	}
	var activeCPs []models.CampaignPlayer
	for _, cp := range allCPs {
		if cp.Role == models.RolePlayer && cp.Status == models.StatusActive && cp.Campaign != nil {
			activeCPs = append(activeCPs, cp)
		}
	}
	if len(activeCPs) == 0 {
		respondWithTokenFile(s, i, outDisk, name, fmt.Sprintf(messages.TokenApplySuccess, name), homeRow)
		return
	}

	var options []discordgo.SelectMenuOption
	for _, cp := range activeCPs {
		options = append(options, discordgo.SelectMenuOption{
			Label: cp.Campaign.Name,
			Value: cp.CampaignID,
		})
	}

	respondWithTokenFile(s, i, outDisk, name, fmt.Sprintf(messages.TokenPostcreateHeader, name), []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    fmt.Sprintf("%s:%s", messages.TokenPostcreateSelectPrefix, media.ID),
				Placeholder: messages.TokenPostcreateSelectPlaceholder,
				Options:     options,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.TokenSkipLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.TokenSkipPrefix, media.ID),
			},
		}},
	})
}

/*
playerTokenPostcreateSelectHandler assigns a freshly-created token to a campaign.

CustomID: player_token_postcreate:{mediaID}
Values[0]: campaignID
*/
type playerTokenPostcreateSelectHandler struct {
	db *bun.DB
}

func (h *playerTokenPostcreateSelectHandler) CustomIDPrefix() string {
	return messages.TokenPostcreateSelectPrefix
}

func (h *playerTokenPostcreateSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil || media.OwnerID != userID {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	_, err = h.db.NewUpdate().Model((*models.CampaignPlayer)(nil)).
		Set("media_id = ?", mediaID).
		Where("player_id = ? AND campaign_id = ?", userID, campaignID).
		Exec(context.Background())
	if err != nil {
		log.Printf("token_postcreate: assign failed player %s campaign %s: %v", userID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	homeRow := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
		}},
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		respondWithTokenFile(s, i, media.Path, media.Name, fmt.Sprintf(messages.TokenPostcreateAssigned, campaignID), homeRow)
		return
	}
	respondWithTokenFile(s, i, media.Path, media.Name, fmt.Sprintf(messages.TokenPostcreateAssigned, campaign.Name), homeRow)
}

/*
playerTokenSkipHandler dismisses the post-create assignment prompt.

CustomID: player_token_skip:{mediaID}
*/
type playerTokenSkipHandler struct {
	db *bun.DB
}

func (h *playerTokenSkipHandler) CustomIDPrefix() string { return messages.TokenSkipPrefix }

func (h *playerTokenSkipHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	media, err := db.GetByID[models.Media](h.db, parts[1])
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.TokenSavedNoAssign)
		return
	}
	respondWithTokenFile(s, i, media.Path, media.Name, messages.TokenSavedNoAssign, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.HomeLabel, discordgo.DangerButton, router.ViewMe),
		}},
	})
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

	if helpers.GetUserID(i) != playerID {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	for _, suffix := range []string{"src_", "frm_", "out_"} {
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
			disk, _ := mediaserver.TokenPath(h.dataDir, h.mediaBaseURL, guildID, playerID, suffix+token, ext)
			os.Remove(disk)
		}
	}

	helpers.RespondUpdateTerminal(s, i, messages.TokenDiscardSuccess)
}

// respondWithTokenFile sends the token file as an ephemeral attachment so the user
// can right-click → Save As, without going through the media server.
func respondWithTokenFile(s *discordgo.Session, i *discordgo.InteractionCreate, diskPath, name, content string, components []discordgo.MessageComponent) {
	f, err := os.Open(diskPath)
	if err != nil {
		log.Printf("respondWithTokenFile: open %s: %v", diskPath, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	defer f.Close()

	filename := name
	if filename == "" {
		filename = "token"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Embeds:  []*discordgo.MessageEmbed{},
			Files: []*discordgo.File{{
				Name:        filename + ".png",
				ContentType: "image/png",
				Reader:      f,
			}},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
