package interactions

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

type modalCampaignCreate struct {
	db *bun.DB
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
	if err != nil || slots < 1 {
		respondInteraction(s, i, messages.SlotCountMismatchErrorMessage)
		return
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
	)
	if err != nil {
		log.Printf("%s %v", messages.CampaignCreationFailureErrorMessage, err)
		respondInteraction(s, i, messages.CampaignAndRegistrationFailureErrorMessage)
		return
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
