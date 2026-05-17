package commands

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
nextSessionsCommand shows the invoking player's upcoming sessions as a slash command.

The same data is available via the /me hub -> Next Sessions button.
This command offers a direct shortcut without navigating the hub.
*/
type nextSessionsCommand struct {
	db *bun.DB
}

func (c *nextSessionsCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.NextSessionsCommandName,
		Description: messages.NextSessionsCommandDesc,
	}
}

func (c *nextSessionsCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	entries, err := models.GetPlayerCampaigns(c.db, userID)
	if err != nil {
		log.Printf("nextsessions: load campaigns for %s: %v", userID, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	now := time.Now().UTC()

	type upcoming struct {
		Name string
		When time.Time
	}
	var list []upcoming
	for _, e := range entries {
		if e.Status != models.StatusActive {
			continue
		}
		if e.Campaign == nil || !e.Campaign.IsApproved {
			continue
		}
		if e.Campaign.Schedule.NextSession.IsZero() || !e.Campaign.Schedule.NextSession.After(now) {
			continue
		}
		list = append(list, upcoming{
			Name: e.Campaign.Name,
			When: e.Campaign.Schedule.NextSession.UTC(),
		})
	}

	if len(list) == 0 {
		respond(s, i, messages.NextSessionsNone)
		return
	}

	sort.Slice(list, func(a, b int) bool { return list[a].When.Before(list[b].When) })

	settings, err := models.GetOrCreatePlayerSettings(c.db, userID)
	if err != nil {
		log.Printf("nextsessions: load settings for %s: %v", userID, err)
	}
	loc := time.UTC
	if settings != nil {
		loc = settings.Location()
	}

	var lines []string
	for _, e := range list {
		formatted := helpers.FormatInLocation(e.When, messages.SessionListFormat, loc) + " " + helpers.TZLabel(loc)
		lines = append(lines, fmt.Sprintf("• **%s** — %s · %s", e.Name, formatted, helpers.TimeRemaining(e.When)))
	}
	content := messages.NextSessionsHeader + "\n" + strings.Join(lines, "\n")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
