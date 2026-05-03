package interactions

/*
	Set Next Session flow.

	Step 1: Button click (manage_set_session:<campaignID>) opens a modal with
	        date and time fields.
	Step 2: Modal submit (modal_manage_set_session:<campaignID>) parses the
	        fields, validates them, and writes campaign.NextSession in UTC.

	NextSession is the *specific* upcoming session date — distinct from the
	recurring Schedule (DayOfWeek/StartTime/Frequency). DMs can use this to
	override the schedule for a one-off, or to anchor the next sitting when
	the campaign is freshly approved and has no derived next-date yet.
*/

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
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

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	dateValue := ""
	timeValue := ""
	if !campaign.Schedule.NextSession.IsZero() {
		next := campaign.Schedule.NextSession.UTC()
		dateValue = next.Format(messages.DateInputFormat)
		timeValue = next.Format(messages.TimeInputFormat)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.ManageSetSessionModalID, campaignID),
			Title:    messages.ManageSetSessionModalTitle,
			Components: []discordgo.MessageComponent{
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
						Label:       messages.ManageSetSessionTimeLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: messages.ManageSetSessionTimePlaceholder,
						Value:       timeValue,
					},
				}},
			},
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

	campaign, ok := helpers.LoadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	var dateStr, timeStr string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.ManageSetSessionDateFieldID:
				dateStr = strings.TrimSpace(input.Value)
			case messages.ManageSetSessionTimeFieldID:
				timeStr = strings.TrimSpace(input.Value)
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

	when, err := time.ParseInLocation(messages.DateTimeInputFormat, dateStr+" "+timeStr, time.UTC)
	if err != nil {
		helpers.Respond(s, i, messages.ManageSetSessionInvalidTime)
		return
	}

	if !when.After(time.Now().UTC()) {
		helpers.Respond(s, i, messages.ManageSetSessionInPast)
		return
	}

	campaign.Schedule.NextSession = when
	campaign.Schedule.AlertSent = false
	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_set_session: failed to update campaign %s: %v", campaign.ID, err)
		helpers.Respond(s, i, messages.ManageSetSessionUpdateFailed)
		return
	}
	h.sched.Schedule(i.GuildID, campaign)

	helpers.Respond(s, i, fmt.Sprintf(messages.ManageSetSessionSuccess, campaign.Name, when.Format(messages.SessionTimeFormat), helpers.TimeRemaining(when)))
}
