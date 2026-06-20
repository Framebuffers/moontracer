package interactions

import (
	"fmt"
	"log"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
Flow:
 1. Button (manage_edit_capacity:<campaignID>): opens a modal pre-filled with current Slots.
    Westmarch campaigns get a second optional field for SessionCapacity.
 2. Modal (modal_manage_edit_capacity:<campaignID>): validates the new value is a positive
    integer >= current active player count (DM excluded), saves, and refreshes the billboard.
*/

type manageEditCapacity struct {
	db *bun.DB
}

func (h *manageEditCapacity) CustomIDPrefix() string {
	return messages.ManageEditCapacityPrefix
}

func (h *manageEditCapacity) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	slotsValue := ""
	if campaign.Slots > 0 {
		slotsValue = strconv.Itoa(campaign.Slots)
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.ManageEditCapacitySlotsFieldID,
				Label:       messages.ManageEditCapacitySlotsLabel,
				Style:       discordgo.TextInputShort,
				Placeholder: "e.g. 5",
				Value:       slotsValue,
				Required:    true,
				MaxLength:   4,
			},
		}},
	}

	if campaign.IsWestmarch {
		sessionCapValue := ""
		if campaign.SessionCapacity > 0 {
			sessionCapValue = strconv.Itoa(campaign.SessionCapacity)
		}
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    messages.ManageEditCapacitySessionCapFieldID,
				Label:       messages.ManageEditCapacitySessionCapLabel,
				Style:       discordgo.TextInputShort,
				Placeholder: "e.g. 6",
				Value:       sessionCapValue,
				Required:    false,
				MaxLength:   4,
			},
		}})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   fmt.Sprintf("%s:%s", messages.ManageEditCapacityModalID, campaignID),
			Title:      messages.ManageEditCapacityModalTitle,
			Components: components,
		},
	})
}

type manageEditCapacityModal struct {
	db *bun.DB
}

func (h *manageEditCapacityModal) CustomIDPrefix() string {
	return messages.ManageEditCapacityModalID
}

func (h *manageEditCapacityModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}

	var newSlots int
	var newSessionCap int

	for _, row := range i.ModalSubmitData().Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range ar.Components {
			ti, ok := comp.(*discordgo.TextInput)
			if !ok {
				continue
			}
			switch ti.CustomID {
			case messages.ManageEditCapacitySlotsFieldID:
				v, err := strconv.Atoi(ti.Value)
				if err != nil || v <= 0 {
					helpers.RespondUpdateTerminal(s, i, messages.ManageEditCapacityInvalid)
					return
				}
				newSlots = v
			case messages.ManageEditCapacitySessionCapFieldID:
				if ti.Value == "" {
					continue
				}
				v, err := strconv.Atoi(ti.Value)
				if err != nil || v <= 0 {
					helpers.RespondUpdateTerminal(s, i, messages.ManageEditCapacityInvalid)
					return
				}
				newSessionCap = v
			}
		}
	}

	// Count active non-DM players to enforce the floor.
	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("manage_edit_capacity: failed to load players for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	activeCount := 0
	for _, p := range players {
		if p.Status == models.StatusActive && p.Role != models.RoleDM {
			activeCount++
		}
	}

	if newSlots < activeCount {
		helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageEditCapacityTooLow, activeCount))
		return
	}

	campaign.Slots = newSlots
	if newSessionCap > 0 {
		campaign.SessionCapacity = newSessionCap
	}

	if err := db.Update(h.db, campaign); err != nil {
		log.Printf("manage_edit_capacity: update failed for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	go func() {
		if err := helpers.UpdateBillboard(s, h.db, campaign); err != nil {
			log.Printf("manage_edit_capacity: billboard update for %s: %v", campaignID, err)
		}
	}()

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageEditCapacitySuccess, campaign.Name))
}
