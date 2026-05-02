package interactions

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. DM opens `/managecampaigns`, selects a campaign, clicks "Reschedule".
		2. `manageCampaignReschedule` catches `manage_reschedule:<campaignID>`, authorizes (ScopeDM).
		3. Opens a modal with four fields: day of week, start time, duration, frequency.
		4. DM submits the modal, triggering `manage_reschedule_modal:<campaignID>`.
		5. `manageCampaignRescheduleModal` parses and validates all four fields.
		6. Updates the campaign's schedule in the DB via `db.Update`.
		7. Responds to the DM ephemerally: "Schedule updated for X: Saturday 19:00 UTC (3h), weekly."
*/

/*
manageCampaignReschedule opens a modal for the DM to update the campaign schedule.

Custom ID format: manage_reschedule:<campaignID>
*/
type manageCampaignReschedule struct {
	db *bun.DB
}

func (h *manageCampaignReschedule) CustomIDPrefix() string {
	return messages.RescheduleComponentPrefix
}

func (h *manageCampaignReschedule) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := i.Member.User.ID

	ok, err := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID)
	if err != nil || !ok {
		respondInteraction(s, i, messages.ManageNotAuthorized)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: messages.RescheduleModalPrefix + ":" + campaignID,
			Title:    messages.RescheduleModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.RescheduleDayFieldID,
						Label:       messages.RescheduleDayLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: messages.RescheduleDayPlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.RescheduleTimeFieldID,
						Label:       messages.RescheduleTimeLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: messages.RescheduleTimePlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.RescheduleDurFieldID,
						Label:       messages.RescheduleDurLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: messages.RescheduleDurPlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.RescheduleFreqFieldID,
						Label:       messages.RescheduleFreqLabel,
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: messages.RescheduleFreqPlaceholder,
					},
				}},
			},
		},
	})
}

/*
manageCampaignRescheduleModal handles the modal submission and updates the schedule.

Custom ID format: manage_reschedule_modal:<campaignID>
*/
type manageCampaignRescheduleModal struct {
	db *bun.DB
}

func (h *manageCampaignRescheduleModal) CustomIDPrefix() string {
	return messages.RescheduleModalPrefix
}

func (h *manageCampaignRescheduleModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := splitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := loadDMCampaign(s, i, h.db, campaignID)
	if !ok {
		return
	}

	var dayStr, timeStr, durStr, freqStr string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.RescheduleDayFieldID:
				dayStr = strings.TrimSpace(input.Value)
			case messages.RescheduleTimeFieldID:
				timeStr = strings.TrimSpace(input.Value)
			case messages.RescheduleDurFieldID:
				durStr = strings.TrimSpace(input.Value)
			case messages.RescheduleFreqFieldID:
				freqStr = strings.TrimSpace(strings.ToLower(input.Value))
			}
		}
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 0 || day > 6 {
		respondInteraction(s, i, messages.RescheduleInvalidDay)
		return
	}

	if !isValidTime(timeStr) {
		respondInteraction(s, i, messages.RescheduleInvalidTime)
		return
	}

	duration, err := strconv.ParseFloat(durStr, 64)
	if err != nil || duration <= 0 {
		respondInteraction(s, i, messages.RescheduleInvalidDuration)
		return
	}

	freq := models.CampaignFrequency(freqStr)
	if !isValidFrequency(freq) {
		respondInteraction(s, i, messages.RescheduleInvalidFrequency)
		return
	}

	campaign.Schedule.DayOfWeek = day
	campaign.Schedule.StartTime = timeStr
	campaign.Schedule.DurationHours = duration
	campaign.Schedule.Frequency = freq

	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("campaign_reschedule: failed to update schedule: %v", err)
		respondInteraction(s, i, messages.RescheduleError)
		return
	}

	dayName := campaign.Schedule.DayName()
	respondInteraction(s, i, fmt.Sprintf(messages.RescheduleSuccess, campaign.Name, dayName, timeStr, durStr, freqStr))
}

func isValidTime(t string) bool {
	parts := strings.SplitN(t, ":", 2)
	if len(parts) != 2 {
		return false
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return false
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return false
	}
	return true
}

func isValidFrequency(f models.CampaignFrequency) bool {
	switch f {
	case models.Weekly, models.Biweekly, models.Monthly, models.Quarterly, models.Yearly:
		return true
	}
	return false
}
