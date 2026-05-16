package interactions

/*
New-session announcement flow.

Two entry points lead to the same modal:
  - /newsession <campaign> slash command  (newSessionCommand in commands package)
  - "New Session" button in the manage sessions menu (manageNewSessionButton below)

Modal submission (newSessionModal):
  1. Parses date + time in the DM's timezone; notes are optional.
  2. Creates a Session record in the DB.
  3. Posts a public announcement embed in the campaign's channel with going/not going RSVP buttons.
  4. DMs all active members immediately with the same info + buttons.
  5. Calls campaign.RefreshNextSession to keep the legacy NextSession field in sync.
  6. Schedules the 1-hour reminder via the Scheduler.

RSVP handlers (sessionRSVPAcceptHandler / sessionRSVPDeclineHandler):
  - CustomID format: session_rsvp_accept:<guildID>:<sessionID>
  - Works from both the channel embed and reminder DMs (guildID encoded for DM routing).
  - Checks campaign membership, schedule conflicts, and session capacity.
  - Upserts to session_rsvps, edits the channel embed's RSVP counts, DMs the campaign DM.

Conflict confirmation (sessionRSVPConfirmHandler):
  - CustomID: session_rsvp_confirm:<guildID>:<sessionID>
  - Skips the conflict check and writes the RSVP directly.
*/

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/scheduler"
)

/*
	Manage menu button
*/

type manageNewSessionButton struct {
	db *bun.DB
}

func (h *manageNewSessionButton) CustomIDPrefix() string { return messages.ManageNewSessionPrefix }

func (h *manageNewSessionButton) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	settings, _ := models.GetOrCreatePlayerSettings(h.db, userID)
	loc := settings.Location()
	timeLabel := fmt.Sprintf(messages.NewCampaignScheduleTimeLabelFmt, helpers.TZLabel(loc))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   fmt.Sprintf("%s:%s", messages.NewSessionModalID, campaignID),
			Title:      messages.NewSessionModalTitle,
			Components: commands.NewSessionModalRows(timeLabel),
		},
	})
}

/*
	Modal Handler
*/

type newSessionModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
	sched      *scheduler.Scheduler
}

func (h *newSessionModal) CustomIDPrefix() string { return messages.NewSessionModalID }

func (h *newSessionModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)
	guildID := i.GuildID

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("new_session_modal: load settings %s: %v", userID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	loc := settings.Location()

	var dateStr, timeStr, notes string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.NewSessionDateFieldID:
				dateStr = strings.TrimSpace(input.Value)
			case messages.NewSessionTimeFieldID:
				timeStr = strings.TrimSpace(input.Value)
			case messages.NewSessionNotesFieldID:
				notes = strings.TrimSpace(input.Value)
			}
		}
	}

	if _, err := time.Parse(messages.DateInputFormat, dateStr); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.NewCampaignScheduleInvalidDate)
		return
	}
	if !isValidTime(timeStr) {
		helpers.RespondUpdateTerminal(s, i, messages.NewCampaignScheduleInvalidTime)
		return
	}
	when, err := time.ParseInLocation(messages.DateTimeInputFormat, dateStr+" "+timeStr, loc)
	if err != nil || !when.After(time.Now().UTC()) {
		helpers.RespondUpdateTerminal(s, i, messages.NewCampaignScheduleInPast)
		return
	}
	when = when.UTC()

	session := models.NewSession(campaignID, when, notes, 0)
	if _, err := h.db.NewInsert().Model(session).Exec(context.Background()); err != nil {
		log.Printf("new_session_modal: insert session for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// 1. Build the channel announcement embed and RSVP buttons.
	embed := buildSessionEmbed(session, campaign, nil)
	rsvpButtons := sessionRSVPButtons(guildID, session.ID)

	// 2. Post to the campaign channel (announcements thread preferred).
	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}
	if channelID == "" {
		helpers.RespondUpdateTerminal(s, i, messages.NewSessionNoChannel)
		return
	}

	msg, err := guard.ChannelMessageSendComplex(s, channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{rsvpButtons},
	})
	if err != nil {
		log.Printf("new_session_modal: post to channel %s: %v", channelID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	// 3. Save the announcement message ID so RSVP handlers can edit it later.
	session.ChannelMsgID = msg.ID
	if _, err := h.db.NewUpdate().Model(session).Column("channel_msg_id").WherePK().Exec(context.Background()); err != nil {
		log.Printf("new_session_modal: save msg_id for session %s: %v", session.ID, err)
	}

	// 4. DM all active members immediately.
	players, _ := models.GetCampaignPlayers(h.db, campaignID)
	notesBlock := ""
	if notes != "" {
		notesBlock = "\n" + notes
	}
	linkBlock := buildAnnounceLinkBlock(campaign, "")
	msgID := uuid.NewString()
	sent := 0
	for _, p := range players {
		if p.Status != models.StatusActive {
			continue
		}
		if p.PlayerID == userID {
			continue // NOTE: don't DM the DM who just set this up
		}
		pSettings, _ := models.GetOrCreatePlayerSettings(h.db, p.PlayerID)
		if !pSettings.NotifySessionRemind {
			continue
		}
		content := fmt.Sprintf(messages.NewSessionDMContentFmt,
			campaign.Name,
			when.Unix(), when.Unix(),
			notesBlock,
		) + linkBlock
		h.dispatcher.Push(dispatch.DirectMessage{
			ID:      msgID,
			Target:  p.PlayerID,
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.SessionRSVPAcceptLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPAcceptPrefix, guildID, session.ID),
					},
					discordgo.Button{
						Label:    messages.SessionRSVPDeclineLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPDeclinePrefix, guildID, session.ID),
					},
				}},
			},
		})
		sent++
	}

	// Refresh campaign.NextSession cache and schedule the 1-hour reminder.
	if err := campaign.RefreshNextSession(h.db); err != nil {
		log.Printf("new_session_modal: refresh next session for %s: %v", campaignID, err)
	}
	h.sched.ScheduleSession(guildID, session)

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.NewSessionAnnouncedFmt, campaign.Name, sent),
		[]*discordgo.MessageEmbed{}, nil)
}

/*
	RSVP handlers
*/

type sessionRSVPAcceptHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionRSVPAcceptHandler) CustomIDPrefix() string { return messages.SessionRSVPAcceptPrefix }
func (h *sessionRSVPAcceptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleSessionRSVP(s, i, h.db, h.dispatcher, models.RSVPAccepted, false)
}

type sessionRSVPDeclineHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionRSVPDeclineHandler) CustomIDPrefix() string { return messages.SessionRSVPDeclinePrefix }
func (h *sessionRSVPDeclineHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleSessionRSVP(s, i, h.db, h.dispatcher, models.RSVPDeclined, false)
}

type sessionRSVPConfirmHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionRSVPConfirmHandler) CustomIDPrefix() string { return messages.SessionRSVPConfirmPrefix }
func (h *sessionRSVPConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// NOTE: Confirm despite conflict: skip the conflict check.
	handleSessionRSVP(s, i, h.db, h.dispatcher, models.RSVPAccepted, true)
}

type sessionRSVPCancelHandler struct{}

func (h *sessionRSVPCancelHandler) CustomIDPrefix() string { return messages.SessionRSVPCancelPrefix }
func (h *sessionRSVPCancelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	helpers.RespondUpdate(s, i, "ℹ️ RSVP cancelled.", []*discordgo.MessageEmbed{}, nil)
}

/*
	Core logic
*/

func handleSessionRSVP(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	guildDB *bun.DB,
	dispatcher *dispatch.Dispatcher,
	intent models.RSVPStatus, // RSVPAccepted or RSVPDeclined
	skipConflictCheck bool,
) {
	// CustomID: prefix:<guildID>:<sessionID>
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	sessionID := parts[2]
	playerID := helpers.GetUserID(i)
	fromDM := i.GuildID == ""

	ctx := context.Background()

	// 1. Load session.
	session := &models.Session{}
	if err := guildDB.NewSelect().Model(session).Where("id = ?", sessionID).Scan(ctx); err != nil {
		helpers.RespondUpdate(s, i, messages.SessionRSVPGone, []*discordgo.MessageEmbed{}, nil)
		return
	}
	if session.Status != models.SessionUpcoming {
		helpers.RespondUpdate(s, i, messages.SessionRSVPGone, []*discordgo.MessageEmbed{}, nil)
		return
	}

	// 2. Load campaign.
	campaign, err := db.GetByID[models.Campaign](guildDB, session.CampaignID)
	if err != nil || campaign.IsArchived {
		helpers.RespondUpdate(s, i, messages.RSVPCampaignGone, []*discordgo.MessageEmbed{}, nil)
		return
	}

	// 3. check if it is an active campaign member.
	cp := &models.CampaignPlayer{}
	if err := guildDB.NewSelect().Model(cp).
		Where("player_id = ? AND campaign_id = ?", playerID, session.CampaignID).
		Scan(ctx); err != nil || cp.Status != models.StatusActive {
		helpers.RespondUpdate(s, i, messages.SessionRSVPNotMember, []*discordgo.MessageEmbed{}, nil)
		return
	}

	// 4. check idempotency: has the player responded?
	existing, _ := models.GetPlayerSessionRSVP(guildDB, sessionID, playerID)
	if existing != nil && existing.Status != models.RSVPPending {
		helpers.RespondUpdate(s, i, messages.SessionRSVPAlreadySet, []*discordgo.MessageEmbed{}, nil)
		return
	}

	finalStatus := intent

	if intent == models.RSVPAccepted {
		// a. conflict check.
		if !skipConflictCheck {
			conflicts, _ := models.GetPlayerConflictingSessions(guildDB, playerID, session.ScheduledAt)
			if len(conflicts) > 0 {
				conflict := conflicts[0]
				// b. load campaign name for the conflicting session.
				conflictCampaign, _ := db.GetByID[models.Campaign](guildDB, conflict.CampaignID)
				conflictName := conflict.CampaignID
				if conflictCampaign != nil {
					conflictName = conflictCampaign.Name
				}
				pSettings, _ := models.GetOrCreatePlayerSettings(guildDB, playerID)
				conflictTime := helpers.FormatInLocation(conflict.ScheduledAt, messages.SessionListFormat, pSettings.Location())

				guildID := parts[1]
				confirmID := fmt.Sprintf("%s:%s:%s", messages.SessionRSVPConfirmPrefix, guildID, sessionID)
				cancelID := fmt.Sprintf("%s:%s:%s", messages.SessionRSVPCancelPrefix, guildID, sessionID)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(messages.SessionRSVPConflictFmt, conflictName, conflictTime),
						Flags:   discordgo.MessageFlagsEphemeral,
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    messages.SessionRSVPConfirmLabel,
									Style:    discordgo.DangerButton,
									CustomID: confirmID,
								},
								discordgo.Button{
									Label:    messages.SessionRSVPCancelLabel,
									Style:    discordgo.SecondaryButton,
									CustomID: cancelID,
								},
							}},
						},
					},
				})
				return
			}
		}

		// 5. Capacity check: if session has a cap (or falls back to campaign's SessionCapacity).
		cap := session.Capacity
		if cap == 0 {
			cap = campaign.SessionCapacity
		}
		if cap > 0 {
			accepted, _ := models.CountAcceptedRSVPs(guildDB, sessionID)
			if accepted >= cap {
				finalStatus = models.RSVPWaitlisted
			}
		}
	}

	// 6. Write RSVP.
	if err := models.UpsertSessionRSVP(guildDB, sessionID, playerID, finalStatus); err != nil {
		log.Printf("session_rsvp: upsert for %s/%s: %v", playerID, sessionID, err)
		helpers.RespondUpdate(s, i, messages.GenericErrorMessage, []*discordgo.MessageEmbed{}, nil)
		return
	}

	// 7. Player confirmation text.
	var confirmMsg string
	switch finalStatus {
	case models.RSVPAccepted:
		confirmMsg = messages.SessionRSVPAcceptedMsg
	case models.RSVPDeclined:
		confirmMsg = messages.SessionRSVPDeclinedMsg
	default:
		confirmMsg = messages.SessionRSVPWaitlistedMsg
	}

	// 8. Update channel announcement embed with new RSVP counts.
	rsvps, _ := models.GetSessionRSVPs(guildDB, sessionID)
	newEmbed := buildSessionEmbed(session, campaign, rsvps)
	guildID := parts[1]
	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}
	if channelID != "" && session.ChannelMsgID != "" {
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      session.ChannelMsgID,
			Channel: channelID,
			Embeds:  &[]*discordgo.MessageEmbed{newEmbed},
			Components: &[]discordgo.MessageComponent{
				sessionRSVPButtons(guildID, sessionID),
			},
		}); err != nil {
			log.Printf("session_rsvp: edit channel embed %s: %v", session.ChannelMsgID, err)
		}
	}

	// 9. Respond: if from DM, update the DM message; if from channel, update the embed in-place.
	if fromDM {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    i.Message.Content + "\n\n" + confirmMsg,
				Components: []discordgo.MessageComponent{},
			},
		})
	} else {
		/*
			Update the channel embed in-place (the edit above already updated counts).
			Then, show a brief ephemeral confirmation to the player.
		*/
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: confirmMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// Notify campaign DM.
	notifyDM(dispatcher, guildDB, campaign, playerID, session, finalStatus)
}

/*
	Helper methods
*/

func buildSessionEmbed(session *models.Session, campaign *models.Campaign, rsvps []models.SessionRSVP) *discordgo.MessageEmbed {
	accepted, declined := 0, 0
	for _, r := range rsvps {
		switch r.Status {
		case models.RSVPAccepted:
			accepted++
		case models.RSVPDeclined:
			declined++
		}
	}

	desc := fmt.Sprintf("<t:%d:F> · <t:%d:R>", session.ScheduledAt.Unix(), session.ScheduledAt.Unix())
	if session.Title != "" {
		desc += "\n" + session.Title
	}
	desc += "\n\n" + fmt.Sprintf(messages.SessionEmbedGoingFmt, accepted, declined)

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📅 New Session — %s", campaign.Name),
		Description: desc,
		Color:       messages.EmbedColor,
	}
}

func sessionRSVPButtons(guildID, sessionID string) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.SessionRSVPAcceptLabel,
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPAcceptPrefix, guildID, sessionID),
		},
		discordgo.Button{
			Label:    messages.SessionRSVPDeclineLabel,
			Style:    discordgo.DangerButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPDeclinePrefix, guildID, sessionID),
		},
	}}
}

func notifyDM(dispatcher *dispatch.Dispatcher, guildDB *bun.DB, campaign *models.Campaign, playerID string, session *models.Session, status models.RSVPStatus) {
	if campaign.DungeonMaster == "" {
		return
	}
	dmSettings, err := models.GetOrCreatePlayerSettings(guildDB, campaign.DungeonMaster)
	if err != nil {
		return
	}
	sessionTime := helpers.FormatInLocation(session.ScheduledAt, messages.SessionTimeFormat, dmSettings.Location()) +
		" " + helpers.TZLabel(dmSettings.Location())

	var notifyFmt string
	switch status {
	case models.RSVPAccepted:
		notifyFmt = messages.SessionRSVPDMNotifyAccept
	case models.RSVPDeclined:
		notifyFmt = messages.SessionRSVPDMNotifyDecline
	default:
		notifyFmt = messages.SessionRSVPDMNotifyWaitlist
	}

	dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("session-rsvp-notify:%s:%s:%s", session.ID, playerID, string(status)),
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(notifyFmt, playerID, campaign.Name, sessionTime),
	})
}
