package interactions

/*
	Admin Broadcast handler for the /admin hub.

	Flow:
		1. Staff clicks "Broadcast" on the /admin hub.
		2. Auth: ScopeMod.
		3. Open a modal with a paragraph text field for the broadcast message.
		4. On submit, load all registered Players and push a DM via the
		   dispatcher to every one (excluding the sender unless DEBUG_ADMIN_ID
		   matches, mirroring the per-campaign announce flow).
		5. Confirm with the count of dispatched messages.
*/

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

type adminBroadcastHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *adminBroadcastHandler) CustomIDPrefix() string {
	return messages.AdminBroadcastPrefix
}

func (h *adminBroadcastHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.AdminBroadcastModalID,
			Title:    messages.AdminBroadcastModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: messages.AdminBroadcastFieldID,
						Label:    messages.AdminBroadcastFieldLabel,
						Style:    discordgo.TextInputParagraph,
						Required: true,
					},
				}},
			},
		},
	})
}

type adminBroadcastModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *adminBroadcastModal) CustomIDPrefix() string {
	return messages.AdminBroadcastModalID
}

func (h *adminBroadcastModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)

	ok, err := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if err != nil || !ok {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignDBNotStaff)
		return
	}

	var message string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			if input.CustomID == messages.AdminBroadcastFieldID {
				message = input.Value
			}
		}
	}

	if message == "" {
		helpers.RespondUpdateTerminal(s, i, messages.AdminBroadcastFailed)
		return
	}

	players, err := db.GetAll[models.Player](h.db)
	if err != nil {
		log.Printf("admin_broadcast: failed to load players: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.AdminBroadcastFailed)
		return
	}

	skipSelf := guard.DebugAdminID == "" || guard.DebugAdminID != userID

	msgID := uuid.NewString()
	sent := 0
	for _, p := range players {
		if p.ID == userID && skipSelf {
			continue
		}
		h.dispatcher.Push(dispatch.DirectMessage{
			ID:      msgID,
			Sender:  userID,
			Target:  p.ID,
			Content: fmt.Sprintf(messages.AdminBroadcastDMContent, userID, message),
		})
		sent++
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.AdminBroadcastSent, sent))
}
