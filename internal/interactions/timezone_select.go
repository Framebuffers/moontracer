package interactions

/*
	Timezone preference flow.

	1. User clicks "Timezone" on the /me hub.
	2. timezoneButtonHandler (set_timezone) shows the current timezone and a select menu.
	3. User picks from a curated list of IANA zones.
	4. timezoneSelectHandler (timezone_select) saves to PlayerSettings and confirms.
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
tzOptions is the list shown in the select menu containing all the available timezones.
*/
var tzOptions = []discordgo.SelectMenuOption{
	{Label: "UTC / GMT", Value: "UTC"},
	{Label: "Hawaii (UTC-10)", Value: "Pacific/Honolulu"},
	{Label: "Alaska (UTC-9)", Value: "America/Anchorage"},
	{Label: "Pacific US/Canada (UTC-8/-7)", Value: "America/Los_Angeles"},
	{Label: "Mountain US/Canada (UTC-7/-6)", Value: "America/Denver"},
	{Label: "Central US/Canada (UTC-6/-5)", Value: "America/Chicago"},
	{Label: "Eastern US/Canada (UTC-5/-4)", Value: "America/New_York"},
	{Label: "Venezuela (UTC-4)", Value: "America/Caracas"},
	{Label: "Colombia / Peru (UTC-5)", Value: "America/Bogota"},
	{Label: "Santiago, Chile (UTC-4/-3)", Value: "America/Santiago"},
	{Label: "Brazil / São Paulo (UTC-3/-2)", Value: "America/Sao_Paulo"},
	{Label: "Argentina / Buenos Aires (UTC-3)", Value: "America/Argentina/Buenos_Aires"},
	{Label: "Azores (UTC-1)", Value: "Atlantic/Azores"},
	{Label: "London / Dublin (UTC+0/+1)", Value: "Europe/London"},
	{Label: "Paris / Berlin / Madrid (UTC+1/+2)", Value: "Europe/Paris"},
	{Label: "Helsinki / Kyiv (UTC+2/+3)", Value: "Europe/Helsinki"},
	{Label: "Moscow (UTC+3)", Value: "Europe/Moscow"},
	{Label: "Dubai / UAE (UTC+4)", Value: "Asia/Dubai"},
	{Label: "India (UTC+5:30)", Value: "Asia/Kolkata"},
	{Label: "Thailand / Jakarta (UTC+7)", Value: "Asia/Bangkok"},
	{Label: "China / Singapore (UTC+8)", Value: "Asia/Shanghai"},
	{Label: "Japan / Korea (UTC+9)", Value: "Asia/Tokyo"},
	{Label: "Sydney / Melbourne (UTC+10/+11)", Value: "Australia/Sydney"},
	{Label: "Auckland (UTC+12/+13)", Value: "Pacific/Auckland"},
	{Label: "South Africa (UTC+2)", Value: "Africa/Johannesburg"},
}

type timezoneButtonHandler struct {
	db *bun.DB
}

func (h *timezoneButtonHandler) CustomIDPrefix() string {
	return messages.TimezonePrefix
}

func (h *timezoneButtonHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("timezone: load settings for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	current := settings.Timezone
	if current == "" {
		current = "UTC"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.TimezoneHeader, current),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    messages.TimezoneSelectID,
						MenuType:    discordgo.StringSelectMenu,
						Placeholder: messages.TimezoneSelectPlaceholder,
						Options:     tzOptions,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.BackButton(messages.BackLabel, router.ViewMe),
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

type timezoneSelectHandler struct {
	db *bun.DB
}

func (h *timezoneSelectHandler) CustomIDPrefix() string {
	return messages.TimezoneSelectID
}

func (h *timezoneSelectHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := helpers.GetUserID(i)
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.Respond(s, i, messages.TimezoneInvalid)
		return
	}
	tz := values[0]

	settings, err := models.GetOrCreatePlayerSettings(h.db, userID)
	if err != nil {
		log.Printf("timezone: load settings for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	settings.Timezone = tz
	if _, err := h.db.NewUpdate().Model(settings).WherePK().Exec(context.Background()); err != nil {
		log.Printf("timezone: save for %s: %v", userID, err)
		helpers.Respond(s, i, messages.GenericErrorMessage)
		return
	}

	helpers.Respond(s, i, fmt.Sprintf(messages.TimezoneSuccess, tz))
}
