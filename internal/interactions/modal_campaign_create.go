package interactions

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		1. User submits the /newcampaign modal (3 fields: Name, Max Players, Synopsis & Rules).
		2. modalCampaignCreate parses the form, normalizes the tag from the name
		   (deduping if necessary), and creates the Campaign with IsApproved=false.
		3. Responds with the configuration message: book + format dropdowns and
		   Submit/Cancel buttons. The book/format handlers update the campaign
		   in-place; submit sends the approval DMs to staff (see newcampaign_config.go).
*/

// modalCampaignCreate handles the modal submission from `/newcampaign`.
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

	var name, slotsStr, description string
	for _, row := range data.Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case messages.FieldNameID:
				name = input.Value
			case messages.FieldSlotsID:
				slotsStr = input.Value
			case messages.FieldDescriptionID:
				description = input.Value
			}
		}
	}

	name = strings.TrimSpace(name)
	if name == "" {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignCreationFailureErrorMessage)
		return
	}

	// Slots: empty field -> unlimited (-1). Otherwise must be a positive int (>= 1).
	slots := -1
	slotsStr = strings.TrimSpace(slotsStr)
	if slotsStr != "" {
		parsed, err := strconv.Atoi(slotsStr)
		if err != nil || parsed < 1 {
			helpers.RespondUpdateTerminal(s, i, messages.SlotCountMismatchErrorMessage)
			return
		}
		slots = parsed
	}

	tag, err := models.UniqueTag(m.db, models.NormalizeTag(name))
	if err != nil {
		log.Printf("modal_campaign_create: tag dedup failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.CampaignCreationFailureErrorMessage)
		return
	}

	conf := &models.GameConfig{}
	schedule := &models.CampaignSchedule{}

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
		true,  // open by default
		false, // isOneshot- chosen later via dropdown
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
		helpers.RespondUpdateTerminal(s, i, messages.CampaignAndRegistrationFailureErrorMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf(messages.NewCampaignConfigMessage, created.Name),
			Components: newCampaignConfigComponents(created.ID),
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
