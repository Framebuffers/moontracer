package interactions

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
playerSetSheetHandler opens a modal for the player to set their character sheet URL.

CustomID: player_set_sheet:<campaignID>
*/
type playerSetSheetHandler struct {
	db *bun.DB
}

func (h *playerSetSheetHandler) CustomIDPrefix() string { return messages.PlayerSetSheetPrefix }

func (h *playerSetSheetHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	cp, err := models.GetCampaignPlayer(h.db, userID, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.PlayerSetSheetModalID, campaignID),
			Title:    messages.PlayerSetSheetModalTitle,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.PlayerSetSheetFieldID,
						Label:       messages.PlayerSetSheetFieldLabel,
						Style:       discordgo.TextInputShort,
						Placeholder: messages.PlayerSetSheetFieldPlaceholder,
						Value:       cp.SheetURL,
						Required:    false,
						MaxLength:   500,
					},
				}},
			},
		},
	})
}

/*
playerSetSheetModal saves the character sheet URL to the player's campaign membership.

CustomID: player_set_sheet_modal:<campaignID>
*/
type playerSetSheetModal struct {
	db *bun.DB
}

func (h *playerSetSheetModal) CustomIDPrefix() string { return messages.PlayerSetSheetModalID }

func (h *playerSetSheetModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	cp, err := models.GetCampaignPlayer(h.db, userID, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			if ti, ok := comp.(*discordgo.TextInput); ok && ti.CustomID == messages.PlayerSetSheetFieldID {
				cp.SheetURL = ti.Value
			}
		}
	}

	if err := db.Update(h.db, cp); err != nil {
		log.Printf("player_set_sheet: update failed for player %s campaign %s: %v", userID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.PlayerSetSheetFailed)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.PlayerSetSheetSuccess)
		return
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.PlayerSetSheetSuccess, campaign.Name))
}

/*
playerSetTokenHandler shows the player their available tokens to assign to this campaign.

CustomID: player_set_token:<campaignID>
*/
type playerSetTokenHandler struct {
	db *bun.DB
}

func (h *playerSetTokenHandler) CustomIDPrefix() string { return messages.PlayerSetTokenPrefix }

func (h *playerSetTokenHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	var tokens []*models.Media
	if err := h.db.NewSelect().Model(&tokens).
		Where("owner_id = ? AND kind = ?", userID, models.KindTokenPlayer).
		OrderExpr("created_at DESC").
		Scan(context.Background()); err != nil {
		log.Printf("player_set_token: load tokens for player %s: %v", userID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	if len(tokens) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.PlayerNoTokens)
		return
	}

	if len(tokens) == 1 {
		t := tokens[0]
		assignID := fmt.Sprintf("%s:%s:%s", messages.PlayerTokenAssignPrefix, campaignID, t.ID)
		embed := &discordgo.MessageEmbed{
			Title: "Your Token",
			Color: messages.EmbedColor,
			Image: &discordgo.MessageEmbedImage{URL: t.URL},
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    messages.PlayerTokenAssignLabel,
							Style:    discordgo.SuccessButton,
							CustomID: assignID,
						},
					}},
					helpers.BackRow(router.ViewMyCampaigns),
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// NOTE: when the player has multiple tokens, show a menu
	var options []discordgo.SelectMenuOption
	for _, t := range tokens {
		label := t.Name
		if label == "" {
			label = t.ID[:8]
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: label,
			Value: t.ID,
		})
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: "Pick a token to assign:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    fmt.Sprintf("%s:%s", messages.PlayerTokenSelectPrefix, campaignID),
						Placeholder: messages.PlayerTokenSelectPlaceholder,
						Options:     options,
					},
				}},
				helpers.BackRow(router.ViewMyCampaigns),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
playerTokenSelectHandler handles a token selection from the select menu and shows a preview.

CustomID: player_token_select:<campaignID>
*/
type playerTokenSelectHandler struct {
	db *bun.DB
}

func (h *playerTokenSelectHandler) CustomIDPrefix() string { return messages.PlayerTokenSelectPrefix }

func (h *playerTokenSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	mediaID := values[0]

	media, err := db.GetByID[models.Media](h.db, mediaID)
	if err != nil {
		log.Printf("player_token_select: load media %s: %v", mediaID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	assignID := fmt.Sprintf("%s:%s:%s", messages.PlayerTokenAssignPrefix, campaignID, mediaID)
	embed := &discordgo.MessageEmbed{
		Title: "Token Preview",
		Color: messages.EmbedColor,
		Image: &discordgo.MessageEmbedImage{URL: media.URL},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.PlayerTokenAssignLabel,
						Style:    discordgo.SuccessButton,
						CustomID: assignID,
					},
				}},
				helpers.BackRow(router.ViewMyCampaigns),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
playerTokenAssignHandler assigns the selected token to the player's campaign slot.

CustomID: player_token_assign:<campaignID>:<mediaID>
*/
type playerTokenAssignHandler struct {
	db *bun.DB
}

func (h *playerTokenAssignHandler) CustomIDPrefix() string { return messages.PlayerTokenAssignPrefix }

func (h *playerTokenAssignHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID, mediaID := parts[1], parts[2]
	userID := helpers.GetUserID(i)

	_, err := h.db.NewUpdate().Model((*models.CampaignPlayer)(nil)).
		Set("media_id = ?", mediaID).
		Where("player_id = ? AND campaign_id = ?", userID, campaignID).
		Exec(context.Background())
	if err != nil {
		log.Printf("player_token_assign: update failed for player %s campaign %s: %v", userID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.PlayerTokenAssignSuccess)
		return
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.PlayerTokenAssignSuccess, campaign.Name))
}

/*
playerLeaveConfirmPromptHandler shows a confirmation prompt before leaving a campaign.

CustomID: player_leave_confirm:<campaignID>
*/
type playerLeaveConfirmPromptHandler struct {
	db *bun.DB
}

func (h *playerLeaveConfirmPromptHandler) CustomIDPrefix() string {
	return messages.PlayerLeaveConfirmPrefix
}

func (h *playerLeaveConfirmPromptHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if isDM, _ := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID); isDM {
		helpers.RespondUpdateTerminal(s, i, messages.MasterIsLeavingCampaignErrorMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.PlayerLeaveConfirmMsg, campaign.Name),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.PlayerLeaveConfirmLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s", messages.PlayerLeaveDoPrefix, campaignID),
					},
					router.NavButton(messages.PlayerLeaveCancelLabel, discordgo.SecondaryButton, router.ViewMyCampaigns),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
playerLeaveDoHandler executes the campaign leave after confirmation.

CustomID: player_leave_do:<campaignID>
*/
type playerLeaveDoHandler struct {
	db *bun.DB
}

func (h *playerLeaveDoHandler) CustomIDPrefix() string { return messages.PlayerLeaveDoPrefix }

func (h *playerLeaveDoHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if isDM, _ := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID); isDM {
		helpers.RespondUpdateTerminal(s, i, messages.MasterIsLeavingCampaignErrorMessage)
		return
	}

	if err := models.RemoveCampaignPlayer(h.db, userID, campaign.ID); err != nil {
		log.Printf("player_leave_do: remove player %s from campaign %s: %v", userID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.FailedToLeaveCampaignErrorMessage)
		return
	}

	if campaign.RoleID != "" {
		if err := guard.GuildMemberRoleRemove(s, i.GuildID, userID, campaign.RoleID); err != nil {
			log.Printf("player_leave_do: remove role %s from player %s: %v", campaign.RoleID, userID, err)
		}
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.PlayerLeftCampaignMessage, campaign.Name))
}

/*
playerContactDMHandler opens a modal for the player to send a message to the DM.

CustomID: player_contact_dm:<campaignID>
*/
type playerContactDMHandler struct {
	db *bun.DB
}

func (h *playerContactDMHandler) CustomIDPrefix() string { return messages.PlayerContactDMPrefix }

func (h *playerContactDMHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	if campaign.DungeonMaster == userID {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s:%s", messages.PlayerContactDMModalID, campaignID),
			Title:    fmt.Sprintf(messages.PlayerContactDMModalTitle),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    messages.PlayerContactDMFieldID,
						Label:       messages.PlayerContactDMFieldLabel,
						Style:       discordgo.TextInputParagraph,
						Placeholder: messages.PlayerContactDMFieldPlaceholder,
						Required:    true,
						MaxLength:   1000,
					},
				}},
			},
		},
	})
}

/*
playerContactDMModal delivers the player's message to the DM via DM dispatch.

CustomID: player_contact_dm_modal:<campaignID>
*/
type playerContactDMModal struct {
	db         *bun.DB
	dispatcher *dispatch.Dispatcher
}

func (h *playerContactDMModal) CustomIDPrefix() string { return messages.PlayerContactDMModalID }

func (h *playerContactDMModal) HandleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.ModalSubmitData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.CampaignNotFoundMessage)
		return
	}

	var message string
	for _, row := range i.ModalSubmitData().Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			if ti, ok := comp.(*discordgo.TextInput); ok && ti.CustomID == messages.PlayerContactDMFieldID {
				message = ti.Value
			}
		}
	}

	h.dispatcher.Push(dispatch.DirectMessage{
		ID:      fmt.Sprintf("contact:%s:%s", campaignID, userID),
		Target:  campaign.DungeonMaster,
		Content: fmt.Sprintf(messages.PlayerContactDMReceived, userID, campaign.Name, message),
	})

	helpers.RespondUpdateTerminal(s, i, messages.PlayerContactDMSuccess)
}
