package interactions

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow (post-modal new-campaign config):
		1. modalCampaignCreate creates the Campaign in DB (IsApproved=false) and
		   shows a follow-up message with two dropdowns (book + format) and two
		   buttons (Submit / Cancel).
		2. Each dropdown selection updates the campaign in place via Update.
		3. Submit sends approval DMs to staff and confirms to the creator.
		4. Cancel deletes the campaign and its CampaignPlayer entries.

	CustomID format:
		newcampaign_book:<campaignID>
		newcampaign_format:<campaignID>
		newcampaign_submit:<campaignID>
		newcampaign_cancel:<campaignID>
*/

/*
newCampaignConfigComponents builds the book/format/frequency dropdowns + submit/cancel buttons.

Reused by the modal create handler and any re-render path.
*/
func newCampaignConfigComponents(campaignID string) []discordgo.MessageComponent {
	bookSelect := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("%s:%s", messages.NewCampaignBookPrefix, campaignID),
		Placeholder: messages.NewCampaignBookPlaceholder,
		Options: []discordgo.SelectMenuOption{
			// note: to add more options, load the label from messages.go, and set the value here.
			{Label: messages.NewCampaignBookLabel5e, Value: "5e"},
			{Label: messages.NewCampaignBookLabel55e, Value: "5.5e"},
			{Label: messages.NewCampaignBookLabelPF2e, Value: "pf2e"},
			{Label: messages.NewCampaignBookLabelOther, Value: "other"},
		},
	}

	formatSelect := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("%s:%s", messages.NewCampaignFormatPrefix, campaignID),
		Placeholder: messages.NewCampaignFormatPlaceholder,
		Options: []discordgo.SelectMenuOption{
			{Label: messages.CampaignLabel, Value: "campaign"},
			{Label: messages.CampaignTypeOneShotLabel, Value: "oneshot"},
			{Label: messages.CampaignTypeWestmarchLabel, Value: "westmarch"},
		},
	}

	freqSelect := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("%s:%s", messages.NewCampaignFreqPrefix, campaignID),
		Placeholder: messages.NewCampaignFreqPlaceholder,
		Options: []discordgo.SelectMenuOption{
			{Label: messages.NewCampaignFreqLabelWeekly, Value: string(models.Weekly)},
			{Label: messages.NewCampaignFreqLabelBiweekly, Value: string(models.Biweekly)},
			{Label: messages.NewCampaignFreqLabelMonthly, Value: string(models.Monthly)},
			{Label: messages.NewCampaignFreqLabelOnce, Value: string(models.Irregular)},
		},
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{bookSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{formatSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{freqSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.NewCampaignSubmitLabel,
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignSubmitPrefix, campaignID),
			},
			discordgo.Button{
				Label:    messages.NewCampaignCancelLabel,
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignCancelPrefix, campaignID),
			},
		}},
	}
}

// parseConfigCustomID extracts the campaign ID from a "<prefix>:<campaignID>" CustomID.
func parseConfigCustomID(customID string) (string, bool) {
	parts := strings.SplitN(customID, ":", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// loadCampaignForConfig fetches the campaign and verifies the invoker is the DM (creator).
func loadCampaignForConfig(database *bun.DB, campaignID, userID string) (*models.Campaign, error) {
	c, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		return nil, err
	}
	if c.DungeonMaster != userID {
		return nil, fmt.Errorf("not authorized")
	}
	return c, nil
}

/*
	Book Select
*/

type newCampaignBookHandler struct {
	db *bun.DB
}

func (h *newCampaignBookHandler) CustomIDPrefix() string { return messages.NewCampaignBookPrefix }

func (h *newCampaignBookHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	c, err := loadCampaignForConfig(h.db, campaignID, helpers.GetUserID(i))
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	c.Game.Edition = values[0]
	if err := db.Update(h.db, c); err != nil {
		log.Printf("newcampaign_book: update failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	/*
		Updating the message for any reason will reset the message.
		Therefore, ack silently and keep going.
	*/
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

/*
	Format Select
*/

type newCampaignFormatHandler struct {
	db *bun.DB
}

func (h *newCampaignFormatHandler) CustomIDPrefix() string { return messages.NewCampaignFormatPrefix }

func (h *newCampaignFormatHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	c, err := loadCampaignForConfig(h.db, campaignID, helpers.GetUserID(i))
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	switch values[0] {
	case "oneshot":
		c.IsOneshot = true
		c.IsWestmarch = false
		c.Schedule.Frequency = models.OneShot
	case "westmarch":
		c.IsOneshot = false
		c.IsWestmarch = true
		c.Schedule.Frequency = models.Westmarch
		c.Slots = math.MaxInt32
	default:
		c.IsOneshot = false
		c.IsWestmarch = false
		c.Schedule.Frequency = models.Weekly
	}

	if err := db.Update(h.db, c); err != nil {
		log.Printf("newcampaign_format: update failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	/*
		Updating the message for any reason will reset the message.
		Therefore, ack silently and keep going.
	*/
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

/*
	Frequency Select
*/

type newCampaignFrequencyHandler struct {
	db *bun.DB
}

func (h *newCampaignFrequencyHandler) CustomIDPrefix() string { return messages.NewCampaignFreqPrefix }

func (h *newCampaignFrequencyHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	c, err := loadCampaignForConfig(h.db, campaignID, helpers.GetUserID(i))
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	c.Schedule.Frequency = models.CampaignFrequency(values[0])
	if err := db.Update(h.db, c); err != nil {
		log.Printf("newcampaign_freq: update failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

/*
	Submit: opens the schedule modal (date + time) before sending approval DMs.
*/

type newCampaignSubmitHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *newCampaignSubmitHandler) CustomIDPrefix() string { return messages.NewCampaignSubmitPrefix }

func (h *newCampaignSubmitHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}

	userID := helpers.GetUserID(i)
	c, err := loadCampaignForConfig(h.db, campaignID, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	if c.Game.Edition == "" || c.Schedule.Frequency == "" {
		helpers.RespondUpdate(s, i, messages.NewCampaignMissingConfigMessage, []*discordgo.MessageEmbed{}, newCampaignConfigComponents(campaignID))
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("newcampaign_submit: load settings for %s: %v", userID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	loc := settings.Location()
	timeLabel := fmt.Sprintf(messages.NewCampaignScheduleTimeLabelFmt, helpers.TZLabel(loc))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignScheduleModalID, campaignID),
			Title:    messages.NewCampaignScheduleModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignScheduleDateFieldID,
						Label:       messages.NewCampaignScheduleDateLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: messages.NewCampaignScheduleDatePlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignScheduleTimeFieldID,
						Label:       timeLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: messages.NewCampaignScheduleTimePlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignWarningsFieldID,
						Label:       messages.NewCampaignWarningsLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: messages.NewCampaignWarningsPlaceholder,
					},
				}},
			},
		},
	})
}

/*
	Schedule Modal: parses optional date/time, saves to Campaign.Schedule.NextSession,
	then sends approval DMs to staff.
*/

type newCampaignScheduleModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *newCampaignScheduleModal) CustomIDPrefix() string {
	return messages.NewCampaignScheduleModalID
}

func (h *newCampaignScheduleModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.ModalSubmitData().CustomID, ":", 2)
	if len(parts) < 2 || parts[1] == "" {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	c, err := loadCampaignForConfig(h.db, campaignID, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("newcampaign_schedule: load settings for %s: %v", userID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	loc := settings.Location()

	var dateStr, timeStr, warningsStr string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.NewCampaignScheduleDateFieldID:
				dateStr = strings.TrimSpace(input.Value)
			case messages.NewCampaignScheduleTimeFieldID:
				timeStr = strings.TrimSpace(input.Value)
			case messages.NewCampaignWarningsFieldID:
				warningsStr = strings.TrimSpace(input.Value)
			}
		}
	}

	// Parse warnings: split on comma or newline, trim, drop empties.
	if warningsStr != "" {
		for _, w := range strings.FieldsFunc(warningsStr, func(r rune) bool { return r == ',' || r == '\n' }) {
			if w = strings.TrimSpace(w); w != "" {
				c.Warnings = append(c.Warnings, w)
			}
		}
	}

	needsSave := len(c.Warnings) > 0

	if dateStr != "" || timeStr != "" {
		if dateStr == "" || timeStr == "" {
			msg := messages.NewCampaignScheduleInvalidDate
			if dateStr != "" {
				msg = messages.NewCampaignScheduleInvalidTime
			}
			helpers.RespondUpdateTerminal(s, i, msg)
			return
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
		if err != nil {
			helpers.RespondUpdateTerminal(s, i, messages.NewCampaignScheduleInvalidTime)
			return
		}
		if !when.After(time.Now().UTC()) {
			helpers.RespondUpdateTerminal(s, i, messages.NewCampaignScheduleInPast)
			return
		}
		c.Schedule.NextSession = when.UTC()
		needsSave = true
	}

	if needsSave {
		if err := db.Update(h.db, c); err != nil {
			log.Printf("newcampaign_schedule: update failed: %v", err)
			helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
			return
		}
		if c.IsApproved && c.BillboardThreadID != "" {
			go func() {
				if err := helpers.UpdateBillboard(s, h.db, c); err != nil {
					log.Printf("newcampaign_schedule: billboard update for %s: %v", c.ID, err)
				}
			}()
		}
	}

	// Show a bridge message with two buttons: open game-details modal or submit directly.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messages.NewCampaignGameDetailsPrompt,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.NewCampaignGameDetailsOpenLabel,
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignGameDetailsOpenPrefix, c.ID),
					},
					discordgo.Button{
						Label:    messages.NewCampaignSubmitApprovalLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignSubmitApprovalPrefix, c.ID),
					},
				}},
			},
		},
	})
}

/*
	Cancellation
*/

type newCampaignCancelHandler struct {
	db *bun.DB
}

func (h *newCampaignCancelHandler) CustomIDPrefix() string { return messages.NewCampaignCancelPrefix }

func (h *newCampaignCancelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}

	c, err := loadCampaignForConfig(h.db, campaignID, helpers.GetUserID(i))
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	ctx := context.Background()
	if _, err := h.db.NewDelete().Model((*models.CampaignPlayer)(nil)).
		Where("campaign_id = ?", c.ID).Exec(ctx); err != nil {
		log.Printf("newcampaign_cancel: failed to delete campaign players: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if err := db.Delete[models.Campaign](h.db, c.ID); err != nil {
		log.Printf("newcampaign_cancel: failed to delete campaign: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	helpers.RespondUpdate(s, i, messages.NewCampaignCancelledMessage, nil, nil)
}

/*
dispatchApprovalRequest sends approval DMs to all staff and responds to the interaction
with a submitted confirmation.

Used by both the submit-approval button and game-details modal.
*/
func dispatchApprovalRequest(d *dispatch.Dispatcher, database *bun.DB, s *discordgo.Session, i *discordgo.InteractionCreate, c *models.Campaign, userID string) {
	staffMembers, err := db.GetStaff(database)
	if err != nil {
		log.Printf("newcampaign: failed to get staff for campaign %s: %v", c.ID, err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignStaffNotifyFailureMessage)
		return
	}

	guildID := i.GuildID
	approvalButtons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.ApproveButtonLabel,
					Style:    discordgo.SuccessButton,
					CustomID: messages.CampaignApprovePrefix + ":" + guildID + ":" + c.ID,
				},
				discordgo.Button{
					Label:    messages.DenyButtonLabel,
					Style:    discordgo.DangerButton,
					CustomID: messages.CampaignDenyPrefix + ":" + guildID + ":" + c.ID,
				},
			},
		},
	}

	players, _ := models.GetCampaignPlayers(database, c.ID)
	coverURL := models.CoverURLForCampaign(database, c.ID)
	campaignEmbed := commands.CampaignEmbed(*c, players, coverURL, "", userID)

	msgID := uuid.NewString()
	for _, staff := range staffMembers {
		d.Push(dispatch.DirectMessage{
			ID:         msgID,
			Sender:     userID,
			Target:     staff.ID,
			Content:    fmt.Sprintf(messages.CampaignApprovalRequestMessage, c.Name, userID),
			Components: approvalButtons,
			Embeds:     []*discordgo.MessageEmbed{campaignEmbed},
		})
	}

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.NewCampaignSubmittedMessage, c.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		helpers.BackRow(router.ViewManage),
	})
}

/*
	Game Details Open Button: opens the game-details modal from the bridge message.
*/

type newCampaignGameDetailsOpenHandler struct {
	db *bun.DB
}

func (h *newCampaignGameDetailsOpenHandler) CustomIDPrefix() string {
	return messages.NewCampaignGameDetailsOpenPrefix
}

func (h *newCampaignGameDetailsOpenHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	if _, err := loadCampaignForConfig(h.db, campaignID, helpers.GetUserID(i)); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.NewCampaignGameDetailsModalID, campaignID),
			Title:    messages.NewCampaignGameDetailsModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignRulesFieldID,
						Label:       messages.NewCampaignRulesLabel,
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: messages.NewCampaignRulesPlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignVTTFieldID,
						Label:       messages.NewCampaignVTTLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: messages.NewCampaignVTTPlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignBooksFieldID,
						Label:       messages.NewCampaignBooksLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: messages.NewCampaignBooksPlaceholder,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.NewCampaignExtraFieldID,
						Label:       messages.NewCampaignExtraLabel,
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: messages.NewCampaignExtraPlaceholder,
					},
				}},
			},
		},
	})
}

/*
	Submit Approval Button: skips game details and sends approval DMs directly.
*/

type newCampaignSubmitApprovalHandler struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *newCampaignSubmitApprovalHandler) CustomIDPrefix() string {
	return messages.NewCampaignSubmitApprovalPrefix
}

func (h *newCampaignSubmitApprovalHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	campaignID, ok := parseConfigCustomID(i.MessageComponentData().CustomID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	userID := helpers.GetUserID(i)
	c, err := loadCampaignForConfig(h.db, campaignID, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}
	dispatchApprovalRequest(h.dispatcher, h.db, s, i, c, userID)
}

/*
	Game Details Modal: parses rules, VTT platform, books allowed, and extra info,
	saves them to the campaign, then sends approval DMs to staff.
*/

type newCampaignGameDetailsModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *newCampaignGameDetailsModal) CustomIDPrefix() string {
	return messages.NewCampaignGameDetailsModalID
}

func (h *newCampaignGameDetailsModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.ModalSubmitData().CustomID, ":", 2)
	if len(parts) < 2 || parts[1] == "" {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	c, err := loadCampaignForConfig(h.db, campaignID, userID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	var rulesStr, vttStr, booksStr, extraStr string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.NewCampaignRulesFieldID:
				rulesStr = strings.TrimSpace(input.Value)
			case messages.NewCampaignVTTFieldID:
				vttStr = strings.TrimSpace(input.Value)
			case messages.NewCampaignBooksFieldID:
				booksStr = strings.TrimSpace(input.Value)
			case messages.NewCampaignExtraFieldID:
				extraStr = strings.TrimSpace(input.Value)
			}
		}
	}

	c.Game.Rules = rulesStr
	c.Game.VTT = vttStr
	c.Extra = extraStr
	if booksStr != "" {
		for _, b := range strings.FieldsFunc(booksStr, func(r rune) bool { return r == ',' || r == '\n' }) {
			if b = strings.TrimSpace(b); b != "" {
				c.Game.BooksAllowed = append(c.Game.BooksAllowed, b)
			}
		}
	}

	if rulesStr != "" || vttStr != "" || booksStr != "" || extraStr != "" {
		if err := db.Update(h.db, c); err != nil {
			log.Printf("newcampaign_gamedetails: update failed: %v", err)
			helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
			return
		}
	}

	dispatchApprovalRequest(h.dispatcher, h.db, s, i, c, userID)
}
