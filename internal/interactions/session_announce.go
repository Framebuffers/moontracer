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
		helpers.Respond(s, i, messages.GenericErrorMessage)
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
		helpers.Respond(s, i, messages.NewCampaignScheduleInvalidDate)
		return
	}
	if !isValidTime(timeStr) {
		helpers.Respond(s, i, messages.NewCampaignScheduleInvalidTime)
		return
	}
	when, err := time.ParseInLocation(messages.DateTimeInputFormat, dateStr+" "+timeStr, loc)
	if err != nil || !when.After(time.Now().UTC()) {
		helpers.Respond(s, i, messages.NewCampaignScheduleInPast)
		return
	}
	when = when.UTC()

	session := models.NewSession(campaignID, when, notes, 0)
	if _, err := h.db.NewInsert().Model(session).Exec(context.Background()); err != nil {
		log.Printf("new_session_modal: insert session for %s: %v", campaignID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
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
		helpers.Respond(s, i, messages.NewSessionNoChannel)
		return
	}

	var roleMention string
	if campaign.RoleID != "" {
		roleMention = fmt.Sprintf("<@&%s>", campaign.RoleID)
	}
	msg, err := guard.ChannelMessageSendComplex(s, channelID, &discordgo.MessageSend{
		Content:    roleMention,
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{rsvpButtons},
	})
	if err != nil {
		log.Printf("new_session_modal: post to channel %s: %v", channelID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
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

	helpers.Respond(s, i, fmt.Sprintf(messages.NewSessionAnnouncedFmt, campaign.Name, sent))
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
	helpers.RespondUpdate(s, i, messages.SessionRSVPCancelledMsg, []*discordgo.MessageEmbed{}, nil)
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

	/*
		For channel interactions, errors must be ephemeral so the embed stays intact.
		For DM interactions, update the DM message in place.
	*/
	rsvpError := func(msg string) {
		if fromDM {
			helpers.RespondUpdate(s, i, msg, []*discordgo.MessageEmbed{}, nil)
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}

	ctx := context.Background()

	// 1. Load session.
	session := &models.Session{}
	if err := guildDB.NewSelect().Model(session).Where("id = ?", sessionID).Scan(ctx); err != nil {
		rsvpError(messages.SessionRSVPGone)
		return
	}
	if session.Status != models.SessionUpcoming {
		rsvpError(messages.SessionRSVPGone)
		return
	}

	// 2. Load campaign.
	campaign, err := db.GetByID[models.Campaign](guildDB, session.CampaignID)
	if err != nil || campaign.IsArchived {
		rsvpError(messages.RSVPCampaignGone)
		return
	}

	// 3. Must be an active campaign member.
	cp := &models.CampaignPlayer{}
	if err := guildDB.NewSelect().Model(cp).
		Where("player_id = ? AND campaign_id = ?", playerID, session.CampaignID).
		Scan(ctx); err != nil || cp.Status != models.StatusActive {
		rsvpError(messages.SessionRSVPNotMember)
		return
	}

	// 4. Idempotency: already responded?
	existing, _ := models.GetPlayerSessionRSVP(guildDB, sessionID, playerID)
	if existing != nil && existing.Status != models.RSVPPending {
		rsvpError(messages.SessionRSVPAlreadySet)
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
		rsvpError(messages.GenericErrorMessage)
		return
	}

	// 7. Rebuild embed with updated player list.
	rsvps, _ := models.GetSessionRSVPs(guildDB, sessionID)
	newEmbed := buildSessionEmbed(session, campaign, rsvps)
	guildID := parts[1]
	rsvpRow := sessionRSVPButtons(guildID, sessionID)

	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}

	// 8. Respond based on context.
	if fromDM {
		// Update the DM message (remove buttons, show confirmation).
		var confirmMsg string
		switch finalStatus {
		case models.RSVPAccepted:
			confirmMsg = messages.SessionRSVPAcceptedMsg
		case models.RSVPDeclined:
			confirmMsg = messages.SessionRSVPDeclinedMsg
		default:
			confirmMsg = messages.SessionRSVPWaitlistedMsg
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    i.Message.Content + "\n\n" + confirmMsg,
				Components: []discordgo.MessageComponent{},
			},
		})
		// Also update the channel embed so everyone sees the new list.
		if channelID != "" && session.ChannelMsgID != "" {
			if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         session.ChannelMsgID,
				Channel:    channelID,
				Embeds:     &[]*discordgo.MessageEmbed{newEmbed},
				Components: &[]discordgo.MessageComponent{rsvpRow},
			}); err != nil {
				log.Printf("session_rsvp: edit channel embed %s: %v", session.ChannelMsgID, err)
			}
		}
	} else {
		// Update the channel embed in-place — the updated player list is the confirmation.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{newEmbed},
				Components: []discordgo.MessageComponent{rsvpRow},
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
	const maxNames = 10

	var going, notGoing, waitlisted []string
	for _, r := range rsvps {
		mention := "<@" + r.PlayerID + ">"
		switch r.Status {
		case models.RSVPAccepted:
			going = append(going, mention)
		case models.RSVPDeclined:
			notGoing = append(notGoing, mention)
		case models.RSVPWaitlisted:
			waitlisted = append(waitlisted, mention)
		}
	}

	desc := fmt.Sprintf("<t:%d:F> · <t:%d:R>", session.ScheduledAt.Unix(), session.ScheduledAt.Unix())
	if session.Title != "" {
		desc += "\n" + session.Title
	}

	desc += "\n\n" + rsvpLine(messages.SessionEmbedGoingLabel, going, maxNames)
	desc += "\n" + rsvpLine(messages.SessionEmbedNotGoingLabel, notGoing, maxNames)
	if len(waitlisted) > 0 {
		desc += "\n" + rsvpLine(messages.SessionEmbedWaitlistedLabel, waitlisted, maxNames)
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf(messages.SessionEmbedTitleFmt, campaign.Name),
		Description: desc,
		Color:       messages.EmbedColor,
	}
}

// rsvpLine formats "Label (N): @A, @B, +X more" or "Label (0): —".
func rsvpLine(label string, names []string, max int) string {
	if len(names) == 0 {
		return fmt.Sprintf(messages.SessionRSVPLineEmptyFmt, label)
	}
	shown := names
	overflow := 0
	if len(names) > max {
		shown = names[:max]
		overflow = len(names) - max
	}
	line := fmt.Sprintf(messages.SessionRSVPLineFmt, label, len(names), strings.Join(shown, ", "))
	if overflow > 0 {
		line += fmt.Sprintf(messages.SessionRSVPLineOverflowFmt, overflow)
	}
	return line
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
