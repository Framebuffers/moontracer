package interactions

/*
New-session announcement flow.

Two entry points lead to the same modal:
  - /newsession <campaign> slash command  (newSessionCommand in commands package)
  - "New Session" button in the manage sessions menu (manageNewSessionButton below)

Modal submission (newSessionModal):
  1. Parses date + time in the DM's timezone; notes are optional.
  2. Creates a Session record in the DB.
  3. Posts a public announcement embed in the campaign's channel with going/not going response buttons.
  4. DMs all active members immediately with the same info + buttons.
  5. Calls campaign.RefreshNextSession to keep the legacy NextSession field in sync.
  6. Schedules the 1-hour reminder via the Scheduler.

response handlers (sessionResponseAcceptHandler / sessionResponseDeclineHandler):
  - CustomID format: session_response_accept:<guildID>:<sessionID>
  - Works from both the channel embed and reminder DMs (guildID encoded for DM routing).
  - Checks campaign membership, schedule conflicts, and session capacity.
  - Upserts to session_responses, edits the channel embed's response counts, DMs the campaign DM.

Conflict confirmation (sessionResponseConfirmHandler):
  - CustomID: session_response_confirm:<guildID>:<sessionID>
  - Skips the conflict check and writes the response directly.
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

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/cooldown"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
	"github.com/framebuffers/moontracer/internal/scheduler"
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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
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

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
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

	// 1. Build the channel announcement embed and response buttons.
	embed := buildSessionEmbed(session, campaign, nil)
	responseButtons := sessionResponseButtons(guildID, session.ID)

	// 2. Post to the campaign channel (announcements thread preferred).
	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}
	if channelID == "" {
		helpers.Respond(s, i, messages.NewSessionNoChannel)
		return
	}

	send := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{responseButtons},
		/*
			Suppress all mention types by default

			Explicitly allow only the campaign role,
			so a DM can't inject an @everyone mention.
		*/
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	}
	if campaign.RoleID != "" {
		send.Content = fmt.Sprintf("<@&%s>", campaign.RoleID)
		send.AllowedMentions.Roles = []string{campaign.RoleID}
	}
	msg, err := guard.ChannelMessageSendComplex(s, channelID, send)
	if err != nil {
		log.Printf("new_session_modal: post to channel %s: %v", channelID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	// 3. Save the announcement message ID so response handlers can edit it later.
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
		if p.Role == models.RoleDM {
			continue
		}
		if p.PlayerID == userID {
			continue
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
						Label:    messages.SessionResponseAcceptLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionResponseAcceptPrefix, guildID, session.ID),
					},
					discordgo.Button{
						Label:    messages.SessionResponseDeclineLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionResponseDeclinePrefix, guildID, session.ID),
					},
					discordgo.Button{
						Label:    messages.SessionConflictButtonLabel,
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionConflictPrefix, guildID, session.ID),
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
	response handlers
*/

type sessionResponseAcceptHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionResponseAcceptHandler) CustomIDPrefix() string {
	return messages.SessionResponseAcceptPrefix
}
func (h *sessionResponseAcceptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleSessionResponse(s, i, h.db, h.dispatcher, models.ResponseAccepted, false)
}

type sessionResponseDeclineHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionResponseDeclineHandler) CustomIDPrefix() string {
	return messages.SessionResponseDeclinePrefix
}
func (h *sessionResponseDeclineHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handleSessionResponse(s, i, h.db, h.dispatcher, models.ResponseDeclined, false)
}

type sessionResponseConfirmHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionResponseConfirmHandler) CustomIDPrefix() string {
	return messages.SessionResponseConfirmPrefix
}
func (h *sessionResponseConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// NOTE: Confirm despite conflict: skip the conflict check.
	handleSessionResponse(s, i, h.db, h.dispatcher, models.ResponseAccepted, true)
}

type sessionResponseCancelHandler struct{}

func (h *sessionResponseCancelHandler) CustomIDPrefix() string {
	return messages.SessionResponseCancelPrefix
}
func (h *sessionResponseCancelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	helpers.RespondUpdate(s, i, messages.SessionResponseCancelledMsg, []*discordgo.MessageEmbed{}, nil)
}

type sessionResponseRetractHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *sessionResponseRetractHandler) CustomIDPrefix() string {
	return messages.SessionResponseRetractPrefix
}
func (h *sessionResponseRetractHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	guildID := parts[1]
	sessionID := parts[2]
	playerID := helpers.GetUserID(i)
	fromDM := i.GuildID == ""

	responseError := func(msg string) {
		if fromDM {
			helpers.RespondUpdate(s, i, msg, []*discordgo.MessageEmbed{}, nil)
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
			})
		}
	}

	ctx := context.Background()

	session := &models.Session{}
	if err := h.db.NewSelect().Model(session).Where("id = ?", sessionID).Scan(ctx); err != nil {
		responseError(messages.SessionResponseGone)
		return
	}
	if session.Status != models.SessionUpcoming {
		responseError(messages.SessionResponseGone)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, session.CampaignID)
	if err != nil || campaign.IsArchived {
		responseError(messages.ResponseCampaignGone)
		return
	}

	existing, _ := models.GetPlayerSessionConfirmation(h.db, sessionID, playerID)
	if existing == nil || existing.Status == models.ResponsePending {
		responseError(messages.SessionResponseRetractNone)
		return
	}

	if !cooldown.Global.AllowOnce("retract:" + playerID + ":" + sessionID) {
		responseError(messages.SessionResponseRetractUsed)
		return
	}

	wasAccepted := existing.Status == models.ResponseAccepted

	if err := models.UpsertSessionPlayers(h.db, sessionID, playerID, models.ResponsePending); err != nil {
		log.Printf("session_response_retract: upsert for %s/%s: %v", playerID, sessionID, err)
		responseError(messages.GenericErrorMessage)
		return
	}

	// If they had a confirmed seat, promote the first waitlisted player.
	if wasAccepted {
		if next, err := models.GetFirstWaitlistedPlayer(h.db, sessionID); err == nil && next != nil {
			if err := models.UpsertSessionPlayers(h.db, sessionID, next.PlayerID, models.ResponseAccepted); err != nil {
				log.Printf("session_response_retract: promote waitlisted %s: %v", next.PlayerID, err)
			} else {
				sessionTime := session.ScheduledAt.Unix()
				h.dispatcher.Push(dispatch.DirectMessage{
					ID:      uuid.NewString(),
					Target:  next.PlayerID,
					Content: fmt.Sprintf(messages.SessionResponseWaitlistPromoted, campaign.Name, fmt.Sprintf("<t:%d:F>", sessionTime)),
				})
			}
		}
	}

	responses, _ := models.GetSessionConfirmations(h.db, sessionID)
	newEmbed := buildSessionEmbed(session, campaign, responses)
	responseRow := sessionResponseButtons(guildID, sessionID)

	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}

	if fromDM {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    messages.SessionResponseRetractedMsg,
				Components: []discordgo.MessageComponent{},
			},
		})
		if channelID != "" && session.ChannelMsgID != "" {
			if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         session.ChannelMsgID,
				Channel:    channelID,
				Embeds:     &[]*discordgo.MessageEmbed{newEmbed},
				Components: &[]discordgo.MessageComponent{responseRow},
			}); err != nil {
				log.Printf("session_response_retract: edit channel embed %s: %v", session.ChannelMsgID, err)
			}
		}
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: messages.SessionResponseRetractedMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if channelID != "" && session.ChannelMsgID != "" {
			if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         session.ChannelMsgID,
				Channel:    channelID,
				Embeds:     &[]*discordgo.MessageEmbed{newEmbed},
				Components: &[]discordgo.MessageComponent{responseRow},
			}); err != nil {
				log.Printf("session_response_retract: edit channel embed %s: %v", session.ChannelMsgID, err)
			}
		}
	}

	dmSettings, _ := models.GetOrCreatePlayerSettings(h.db, campaign.DungeonMaster)
	sessionTime := helpers.FormatInLocation(session.ScheduledAt, messages.SessionTimeFormat, dmSettings.Location()) +
		" " + helpers.TZLabel(dmSettings.Location())
	h.dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("session-response-retract:%s:%s", session.ID, playerID),
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.SessionResponseDMNotifyRetract, playerID, campaign.Name, sessionTime),
	})
}

/*
	Core logic
*/

func handleSessionResponse(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	guildDB *bun.DB,
	dispatcher *dispatch.Dispatcher,
	intent models.ResponseStatus, // ResponseAccepted or ResponseDeclined
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
	responseError := func(msg string) {
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
		responseError(messages.SessionResponseGone)
		return
	}
	if session.Status != models.SessionUpcoming {
		responseError(messages.SessionResponseGone)
		return
	}

	// 2. Load campaign.
	campaign, err := db.GetByID[models.Campaign](guildDB, session.CampaignID)
	if err != nil || campaign.IsArchived {
		responseError(messages.ResponseCampaignGone)
		return
	}

	// 3. Must be an active campaign member.
	cp := &models.CampaignPlayer{}
	if err := guildDB.NewSelect().Model(cp).
		Where("player_id = ? AND campaign_id = ?", playerID, session.CampaignID).
		Scan(ctx); err != nil || cp.Status != models.StatusActive {
		responseError(messages.SessionResponseNotMember)
		return
	}

	// 4. Cooldown: prevent rapid re-clicks.
	cdKey := "session-response:" + playerID + ":" + sessionID
	if !cooldown.Global.Allow(cdKey, 15*time.Minute) {
		remaining := cooldown.Global.Remaining(cdKey)
		responseError(fmt.Sprintf(messages.SessionResponseCooldown, cooldown.FormatRemaining(remaining)))
		return
	}

	// 5. Idempotency: already responded?
	existing, _ := models.GetPlayerSessionConfirmation(guildDB, sessionID, playerID)
	if existing != nil && existing.Status != models.ResponsePending {
		responseError(messages.SessionResponseAlreadySet)
		return
	}

	finalStatus := intent
	var confirmedConflictName string // set when player confirmed response despite a known clash

	if intent == models.ResponseAccepted {
		// a. conflict check.
		conflicts, _ := models.GetPlayerConflictingSessions(guildDB, playerID, session.ScheduledAt)
		var clashConflicts []models.Session
		for _, c := range conflicts {
			if c.ID != sessionID {
				clashConflicts = append(clashConflicts, c)
			}
		}

		if len(clashConflicts) > 0 && !skipConflictCheck {
			conflict := clashConflicts[0]
			// b. load campaign name for the conflicting session.
			conflictCampaign, _ := db.GetByID[models.Campaign](guildDB, conflict.CampaignID)
			conflictName := conflict.CampaignID
			if conflictCampaign != nil {
				conflictName = conflictCampaign.Name
			}
			pSettings, _ := models.GetOrCreatePlayerSettings(guildDB, playerID)
			conflictTime := helpers.FormatInLocation(conflict.ScheduledAt, messages.SessionListFormat, pSettings.Location())

			guildID := parts[1]
			confirmID := fmt.Sprintf("%s:%s:%s", messages.SessionResponseConfirmPrefix, guildID, sessionID)
			cancelID := fmt.Sprintf("%s:%s:%s", messages.SessionResponseCancelPrefix, guildID, sessionID)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(messages.SessionResponseConflictFmt, conflictName, conflictTime),
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    messages.SessionResponseConfirmLabel,
								Style:    discordgo.DangerButton,
								CustomID: confirmID,
							},
							discordgo.Button{
								Label:    messages.SessionResponseCancelLabel,
								Style:    discordgo.SecondaryButton,
								CustomID: cancelID,
							},
						}},
					},
					AllowedMentions: &discordgo.MessageAllowedMentions{
						Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeRoles},
					},
				},
			})
			return
		}

		// clash confirmed
		if skipConflictCheck && len(clashConflicts) > 0 {
			conflict := clashConflicts[0]
			conflictCampaign, _ := db.GetByID[models.Campaign](guildDB, conflict.CampaignID)
			if conflictCampaign != nil {
				confirmedConflictName = conflictCampaign.Name
			} else {
				confirmedConflictName = conflict.CampaignID
			}
		}

		// 5. Capacity check: if session has a cap (or falls back to campaign's SessionCapacity).
		cap := session.Capacity
		if cap == 0 {
			cap = campaign.SessionCapacity
		}
		if cap > 0 {
			accepted, _ := models.CountAcceptedPlayers(guildDB, sessionID)
			if accepted >= cap {
				finalStatus = models.ResponseWaitlisted
			}
		}
	}

	// 6. Write response.
	if err := models.UpsertSessionPlayers(guildDB, sessionID, playerID, finalStatus); err != nil {
		log.Printf("session_response: upsert for %s/%s: %v", playerID, sessionID, err)
		responseError(messages.GenericErrorMessage)
		return
	}

	// 7. Rebuild embed with updated player list.
	responses, _ := models.GetSessionConfirmations(guildDB, sessionID)
	newEmbed := buildSessionEmbed(session, campaign, responses)
	guildID := parts[1]
	responseRow := sessionResponseButtons(guildID, sessionID)

	channelID := campaign.AnnouncementsThreadID
	if channelID == "" {
		channelID = campaign.ChannelID
	}

	// 8. Respond based on context.
	if fromDM {
		// Update the DM message (remove buttons, show confirmation).
		var confirmMsg string
		switch finalStatus {
		case models.ResponseAccepted:
			confirmMsg = messages.SessionResponseAcceptedMsg
		case models.ResponseDeclined:
			confirmMsg = messages.SessionResponseDeclinedMsg
		default:
			confirmMsg = messages.SessionResponseWaitlistedMsg
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
				Components: &[]discordgo.MessageComponent{responseRow},
			}); err != nil {
				log.Printf("session_response: edit channel embed %s: %v", session.ChannelMsgID, err)
			}
		}
	} else {
		// Update the channel embed in-place- the updated player list is the confirmation.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{newEmbed},
				Components: []discordgo.MessageComponent{responseRow},
			},
		})
	}

	// Notify campaign DM.
	notifyDM(dispatcher, guildDB, campaign, playerID, session, finalStatus, confirmedConflictName)
}

/*
	Helper methods
*/

func buildSessionEmbed(session *models.Session, campaign *models.Campaign, responses []models.SessionAssistance) *discordgo.MessageEmbed {
	const maxNames = 10

	var going, notGoing, waitlisted []string
	for _, r := range responses {
		mention := "<@" + r.PlayerID + ">"
		switch r.Status {
		case models.ResponseAccepted:
			going = append(going, mention)
		case models.ResponseDeclined:
			notGoing = append(notGoing, mention)
		case models.ResponseWaitlisted:
			waitlisted = append(waitlisted, mention)
		}
	}

	desc := fmt.Sprintf("<t:%d:F> · <t:%d:R>", session.ScheduledAt.Unix(), session.ScheduledAt.Unix())
	if session.Title != "" {
		desc += "\n" + session.Title
	}

	desc += "\n\n" + responseLine(messages.SessionEmbedGoingLabel, going, maxNames)
	desc += "\n" + responseLine(messages.SessionEmbedNotGoingLabel, notGoing, maxNames)
	if len(waitlisted) > 0 {
		desc += "\n" + responseLine(messages.SessionEmbedWaitlistedLabel, waitlisted, maxNames)
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf(messages.SessionEmbedTitleFmt, campaign.Name),
		Description: desc,
		Color:       messages.EmbedColor,
	}
}

// responseLine formats "Label (N): @A, @B, +X more" or "Label (0):-".
func responseLine(label string, names []string, max int) string {
	if len(names) == 0 {
		return fmt.Sprintf(messages.SessionResponseLineEmptyFmt, label)
	}
	shown := names
	overflow := 0
	if len(names) > max {
		shown = names[:max]
		overflow = len(names) - max
	}
	line := fmt.Sprintf(messages.SessionResponseLineFmt, label, len(names), strings.Join(shown, ", "))
	if overflow > 0 {
		line += fmt.Sprintf(messages.SessionResponseLineOverflowFmt, overflow)
	}
	return line
}

func sessionResponseButtons(guildID, sessionID string) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.SessionResponseAcceptLabel,
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionResponseAcceptPrefix, guildID, sessionID),
		},
		discordgo.Button{
			Label:    messages.SessionResponseDeclineLabel,
			Style:    discordgo.DangerButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionResponseDeclinePrefix, guildID, sessionID),
		},
		discordgo.Button{
			Label:    messages.SessionResponseRetractLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionResponseRetractPrefix, guildID, sessionID),
		},
	}}
}

func notifyDM(dispatcher *dispatch.Dispatcher, guildDB *bun.DB, campaign *models.Campaign, playerID string, session *models.Session, status models.ResponseStatus, conflictName string) {
	if campaign.DungeonMaster == "" {
		return
	}
	dmSettings, err := models.GetOrCreatePlayerSettings(guildDB, campaign.DungeonMaster)
	if err != nil {
		return
	}
	sessionTime := helpers.FormatInLocation(session.ScheduledAt, messages.SessionTimeFormat, dmSettings.Location()) +
		" " + helpers.TZLabel(dmSettings.Location())

	var content string
	switch status {
	case models.ResponseAccepted:
		if conflictName != "" {
			content = fmt.Sprintf(messages.SessionResponseDMNotifyAcceptConflict, playerID, campaign.Name, sessionTime, conflictName)
		} else {
			content = fmt.Sprintf(messages.SessionResponseDMNotifyAccept, playerID, campaign.Name, sessionTime)
		}
	case models.ResponseDeclined:
		content = fmt.Sprintf(messages.SessionResponseDMNotifyDecline, playerID, campaign.Name, sessionTime)
	default:
		content = fmt.Sprintf(messages.SessionResponseDMNotifyWaitlist, playerID, campaign.Name, sessionTime)
	}

	dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("session-response-notify:%s:%s:%s", session.ID, playerID, string(status)),
		Target:  campaign.DungeonMaster,
		Content: content,
	})
}
