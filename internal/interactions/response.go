package interactions

/*
	Session response flow.

	Players receive a reminder DM ~1 hour before their session with buttons for
	Going or Not Going to a session.

	Clicking either button:
	  1. Saves ResponseStatus on the CampaignPlayer row.
	  2. Updates the reminder message (removes buttons, shows confirmation).
	  3. DMs the campaign DM in their own timezone.

	CustomID format: response_accept:<guildID>:<campaignID>
	                 response_decline:<guildID>:<campaignID>

	Three-part format lets extractGuildFromCustomID resolve the guild for DM interactions.
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Accept
*/

type responseAcceptHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *responseAcceptHandler) CustomIDPrefix() string { return messages.ResponseAcceptPrefix }

func (h *responseAcceptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleResponse(s, i, h.db, h.dispatcher, models.ResponseAccepted)
}

/*
	Decline
*/

type responseDeclineHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *responseDeclineHandler) CustomIDPrefix() string { return messages.ResponseDeclinePrefix }

func (h *responseDeclineHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleResponse(s, i, h.db, h.dispatcher, models.ResponseDeclined)
}

/*
	Handle response
*/

func handleResponse(s *discordgo.Session, i *discordgo.InteractionCreate, guildDB *bun.DB, dispatcher *dispatch.Dispatcher, status models.ResponseStatus) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[2]
	playerID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](guildDB, campaignID)
	if err != nil || campaign.IsArchived {
		helpers.RespondUpdateTerminal(s, i, messages.ResponseCampaignGone)
		return
	}

	var cp models.CampaignPlayer
	err = guildDB.NewSelect().Model(&cp).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Scan(context.Background())
	if err != nil {
		log.Printf("response: load campaign player %s/%s: %v", playerID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if cp.ResponseStatus != models.ResponsePending {
		helpers.RespondUpdateTerminal(s, i, messages.ResponseAlreadyResponded)
		return
	}

	cp.ResponseStatus = status
	if _, err := guildDB.NewUpdate().Model(&cp).Column("response_status").WherePK().Exec(context.Background()); err != nil {
		log.Printf("response: save status for %s/%s: %v", playerID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	confirmText := messages.ResponseAcceptedPlayer
	if status == models.ResponseDeclined {
		confirmText = messages.ResponseDeclinedPlayer
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    i.Message.Content + "\n\n" + confirmText,
			Components: []discordgo.MessageComponent{},
		},
	})

	if campaign.DungeonMaster == "" || campaign.Schedule.NextSession.IsZero() {
		return
	}
	dmSettings, err := models.GetOrCreatePlayerSettings(guildDB, campaign.DungeonMaster)
	if err != nil {
		log.Printf("response: load DM settings for %s: %v", campaign.DungeonMaster, err)
		return
	}
	sessionTime := helpers.FormatInLocation(campaign.Schedule.NextSession, messages.SessionTimeFormat, dmSettings.Location()) +
		" " + helpers.TZLabel(dmSettings.Location())

	notifyFmt := messages.ResponseDMNotifyAccept
	if status == models.ResponseDeclined {
		notifyFmt = messages.ResponseDMNotifyDecline
	}
	dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("response-notify:%s:%s:%s", campaignID, playerID, string(status)),
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(notifyFmt, playerID, campaign.Name, sessionTime),
	})
}
