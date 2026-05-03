package interactions

/*
	Set / Reschedule Session flow.

	Step 1 — Button (manage_set_session:<campaignID>):
		Opens a modal. Title and fields adapt to context:
		  - First set: "Set Session" title, date + time fields.
		  - Re-schedule: "Reschedule Session" title, date + time + optional reason.
		Date/time fields are pre-filled and labelled in the DM's local timezone
		(from PlayerSettings). Input is parsed in that timezone and stored as UTC.

	Step 2 — Modal (modal_manage_set_session:<campaignID>):
		Validates and writes campaign.Schedule.NextSession (UTC).
		On re-schedule with a reason: posts to the campaign's announcements thread
		(if one is set) and writes an audit entry.
*/

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/guard"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
	"moontracer/internal/scheduler"
)

type manageSetSession struct {
	db *bun.DB
}

func (h *manageSetSession) CustomIDPrefix() string {
	return messages.ManageSetSessionPrefix
}

func (h *manageSetSession) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("manage_set_session: load settings for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}
	loc := settings.Location()

	isReschedule := !campaign.Schedule.NextSession.IsZero()

	dateValue, timeValue := "", ""
	if isReschedule {
		next := campaign.Schedule.NextSession.In(loc)
		dateValue = next.Format(messages.DateInputFormat)
		timeValue = next.Format(messages.TimeInputFormat)
	}

	timeLabel := fmt.Sprintf("%s (%s)", messages.ManageSetSessionTimeLabel, helpers.TZLabel(loc))

	modalTitle := messages.ManageSetSessionModalTitle
	if isReschedule {
		modalTitle = messages.ManageRescheduleModalTitle
	}

	rows := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.ManageSetSessionDateFieldID,
				Label:       messages.ManageSetSessionDateLabel,
				Style:       discordgo.TextInputShort,
				Required:    true,
				Placeholder: messages.ManageSetSessionDatePlaceholder,
				Value:       dateValue,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.ManageSetSessionTimeFieldID,
				Label:       timeLabel,
				Style:       discordgo.TextInputShort,
				Required:    true,
				Placeholder: messages.ManageSetSessionTimePlaceholder,
				Value:       timeValue,
			},
		}},
	}
	if isReschedule {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.ManageSetSessionReasonFieldID,
				Label:       messages.ManageSetSessionReasonLabel,
				Style:       discordgo.TextInputParagraph,
				Required:    false,
				Placeholder: messages.ManageSetSessionReasonPlaceholder,
			},
		}})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   fmt.Sprintf("%s:%s", messages.ManageSetSessionModalID, campaignID),
			Title:      modalTitle,
			Components: rows,
		},
	})
}

type manageSetSessionModal struct {
	db    *bun.DB
	sched *scheduler.Scheduler
}

func (h *manageSetSessionModal) CustomIDPrefix() string {
	return messages.ManageSetSessionModalID
}

func (h *manageSetSessionModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
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

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("manage_set_session: load settings for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}
	loc := settings.Location()

	isReschedule := !campaign.Schedule.NextSession.IsZero()

	var dateStr, timeStr, reason string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.ManageSetSessionDateFieldID:
				dateStr = strings.TrimSpace(input.Value)
			case messages.ManageSetSessionTimeFieldID:
				timeStr = strings.TrimSpace(input.Value)
			case messages.ManageSetSessionReasonFieldID:
				reason = strings.TrimSpace(input.Value)
			}
		}
	}

	if _, err := time.Parse(messages.DateInputFormat, dateStr); err != nil {
		helpers.Respond(s, i, messages.ManageSetSessionInvalidDate)
		return
	}
	if !isValidTime(timeStr) {
		helpers.Respond(s, i, messages.ManageSetSessionInvalidTime)
		return
	}

	when, err := time.ParseInLocation(messages.DateTimeInputFormat, dateStr+" "+timeStr, loc)
	if err != nil {
		helpers.Respond(s, i, messages.ManageSetSessionInvalidTime)
		return
	}

	if !when.After(time.Now().UTC()) {
		helpers.Respond(s, i, messages.ManageSetSessionInPast)
		return
	}

	campaign.Schedule.NextSession = when.UTC()
	campaign.Schedule.AlertSent = false
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_set_session: failed to update campaign %s: %v", campaign.ID, err)
		helpers.Respond(s, i, messages.ManageSetSessionUpdateFailed)
		return
	}
	h.sched.Schedule(i.GuildID, campaign)
	if err := models.ResetCampaignRSVPs(h.db, campaign.ID); err != nil {
		log.Printf("manage_set_session: reset RSVPs for %s: %v", campaign.ID, err)
	}

	displayTime := helpers.FormatInLocation(when, messages.SessionTimeFormat, loc) + " " + helpers.TZLabel(loc)
	remaining := helpers.TimeRemaining(when)

	if isReschedule && reason != "" {
		if campaign.AnnouncementsThreadID != "" {
			threadMsg := fmt.Sprintf(messages.ManageSetSessionRescheduleThread, displayTime, reason)
			if _, err := guard.ChannelMessageSend(s, campaign.AnnouncementsThreadID, threadMsg); err != nil {
				log.Printf("manage_set_session: post to thread %s: %v", campaign.AnnouncementsThreadID, err)
			}
		}
		if err := models.InsertAuditEntry(h.db, userID, userID, models.AuditSessionReschedule, reason); err != nil {
			log.Printf("manage_set_session: audit entry for campaign %s: %v", campaign.ID, err)
		}
		helpers.Respond(s, i, fmt.Sprintf(messages.ManageSetSessionRescheduleSuccess, campaign.Name, displayTime, remaining))
		return
	}

	helpers.Respond(s, i, fmt.Sprintf(messages.ManageSetSessionSuccess, campaign.Name, displayTime, remaining))
}
