package interactions

import (
	"moontracer/internal/interactions/helpers"
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
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
newCampaignConfigComponents builds the book/format dropdowns + submit/cancel buttons.

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

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{bookSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{formatSelect}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.SetCoverButtonLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("manage_setcover:%s", campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageLinksLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageLinksPrefix, campaignID),
			},
		}},
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

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.NewCampaignConfigSystemHeader, c.Name, values[0]), nil, newCampaignConfigComponents(campaignID))
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

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.NewCampaignConfigFormatHeader, c.Name, values[0]), nil, newCampaignConfigComponents(campaignID))
}

/*
	Submit.
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

	staffMembers, err := db.GetStaff(h.db)
	if err != nil {
		log.Printf("newcampaign_submit: failed to get staff: %v", err)
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

	msgID := uuid.NewString()
	for _, staff := range staffMembers {
		h.dispatcher.Push(dispatch.DirectMessage{
			ID:         msgID,
			Sender:     userID,
			Target:     staff.ID,
			Content:    fmt.Sprintf(messages.CampaignApprovalRequestMessage, c.Name, userID),
			Components: approvalButtons,
		})
	}

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.NewCampaignSubmittedMessage, c.Name), nil, []discordgo.MessageComponent{
		helpers.BackRow(router.ViewManage),
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
