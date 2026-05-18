package interactions

/*
	Session RSVP flow.

	Players receive a reminder DM ~1 hour before their session with buttons for
	Going or Not Going to a session.

	Clicking either button:
	  1. Saves RSVPStatus on the CampaignPlayer row.
	  2. Updates the reminder message (removes buttons, shows confirmation).
	  3. DMs the campaign DM in their own timezone.

	CustomID format: rsvp_accept:<guildID>:<campaignID>
	                 rsvp_decline:<guildID>:<campaignID>

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

type rsvpAcceptHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *rsvpAcceptHandler) CustomIDPrefix() string { return messages.RSVPAcceptPrefix }

func (h *rsvpAcceptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleRSVP(s, i, h.db, h.dispatcher, models.RSVPAccepted)
}

/*
	Decline
*/

type rsvpDeclineHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *rsvpDeclineHandler) CustomIDPrefix() string { return messages.RSVPDeclinePrefix }

func (h *rsvpDeclineHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleRSVP(s, i, h.db, h.dispatcher, models.RSVPDeclined)
}

/*
	Handle RSVP
*/

func handleRSVP(s *discordgo.Session, i *discordgo.InteractionCreate, guildDB *bun.DB, dispatcher *dispatch.Dispatcher, status models.RSVPStatus) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[2]
	playerID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](guildDB, campaignID)
	if err != nil || campaign.IsArchived {
		helpers.RespondUpdateTerminal(s, i, messages.RSVPCampaignGone)
		return
	}

	var cp models.CampaignPlayer
	err = guildDB.NewSelect().Model(&cp).
		Where("player_id = ? AND campaign_id = ?", playerID, campaignID).
		Scan(context.Background())
	if err != nil {
		log.Printf("rsvp: load campaign player %s/%s: %v", playerID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if cp.RSVPStatus != models.RSVPPending {
		helpers.RespondUpdateTerminal(s, i, messages.RSVPAlreadyResponded)
		return
	}

	cp.RSVPStatus = status
	if _, err := guildDB.NewUpdate().Model(&cp).Column("rsvp_status").WherePK().Exec(context.Background()); err != nil {
		log.Printf("rsvp: save status for %s/%s: %v", playerID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	confirmText := messages.RSVPAcceptedPlayer
	if status == models.RSVPDeclined {
		confirmText = messages.RSVPDeclinedPlayer
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
		log.Printf("rsvp: load DM settings for %s: %v", campaign.DungeonMaster, err)
		return
	}
	sessionTime := helpers.FormatInLocation(campaign.Schedule.NextSession, messages.SessionTimeFormat, dmSettings.Location()) +
		" " + helpers.TZLabel(dmSettings.Location())

	notifyFmt := messages.RSVPDMNotifyAccept
	if status == models.RSVPDeclined {
		notifyFmt = messages.RSVPDMNotifyDecline
	}
	dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("rsvp-notify:%s:%s:%s", campaignID, playerID, string(status)),
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(notifyFmt, playerID, campaign.Name, sessionTime),
	})
}
