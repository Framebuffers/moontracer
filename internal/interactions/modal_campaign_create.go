package interactions

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
	Flow:
		1. User runs `/newcampaign` command.
		2. Bot responds with a modal (form) to fill in campaign details (name, tag, description, edition, slots).
		3. User submits the modal, triggering `modal_campaign_create`.
		4. `modalCampaignCreate` parses form fields, validates inputs (slot count > 0, tag uniqueness).
		5. Creates Campaign in DB with IsApproved=false (pending admin approval).
		6. Finds all users with the ADMIN_ROLE_NAME role and sends them DMs with Approve/Deny buttons.
		7. Responds to creator ephemerally: "Your campaign request has been submitted for admin approval."
*/

// modalCampaignCreate handles the modal submission from `/newcampaign` to create a new campaign.
type modalCampaignCreate struct {
	db       *bun.DB
	dispatch *dispatch.Dispatcher
}

func (m *modalCampaignCreate) CustomIDPrefix() string {
	return messages.NewCampaignModalCustomID
}

func (m *modalCampaignCreate) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID := i.Member.User.ID

	var name, tag, description, edition, slotsStr string
	for _, row := range data.Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.FieldNameID:
				name = input.Value
			case messages.FieldTagID:
				tag = input.Value
			case messages.FieldDescriptionID:
				description = input.Value
			case messages.FieldEditionID:
				edition = input.Value
			case messages.FieldSlotsID:
				slotsStr = input.Value
			}
		}
	}

	slots, err := strconv.Atoi(strings.TrimSpace(slotsStr))
	if err != nil || slots < 0 {
		respondInteraction(s, i, messages.SlotCountMismatchErrorMessage)
		return
	}
	if slots == 0 {
		slots = -1
	}

	conf := &models.GameConfig{
		Edition: edition,
		Rules:   "",
	}
	schedule := &models.CampaignSchedule{
		Frequency: models.Weekly,
	}

	campaign := &models.Campaign{}
	created, err := campaign.CreateCampaign(
		m.db,
		userID,
		nil, // no initial players
		name,
		tag,
		description,
		conf,
		slots,
		true, // open by default
		false,
		nil,
		"",
		schedule,
		nil,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		log.Printf("modal_campaign_create: %s: %v", messages.CampaignCreationFailureErrorMessage, err)
		respondInteraction(s, i, messages.CampaignAndRegistrationFailureErrorMessage)
		return
	}

	// notify mods for approval:
	msgID := uuid.NewString()
	staffMembers, err := db.GetStaff(m.db)
	if err != nil {
		log.Printf("modal_campaign_create: failed to get staff members to notify: %v", err)
		respondInteraction(s, i, messages.CampaignStaffNotifyFailureMessage)
		return
	}

	guildID := i.GuildID
	approvalButtons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    messages.ApproveButtonLabel,
					Style:    discordgo.SuccessButton,
					CustomID: messages.CampaignApprovePrefix + ":" + guildID + ":" + created.ID,
				},
				discordgo.Button{
					Label:    messages.DenyButtonLabel,
					Style:    discordgo.DangerButton,
					CustomID: messages.CampaignDenyPrefix + ":" + guildID + ":" + created.ID,
				},
			},
		},
	}

	for _, staff := range staffMembers {
		m.dispatch.Push(dispatch.DirectMessage{
			ID:         msgID,
			Sender:     userID,
			Target:     staff.ID,
			Content:    fmt.Sprintf(messages.CampaignApprovalRequestMessage, created.Name, userID),
			Components: approvalButtons,
		})
	}

	respondInteraction(s, i, fmt.Sprintf("%s **%s** Use `/campaign tag:%s` to view it.", messages.CampaignCreationMessage, created.Name, created.Tag))
}

func respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
