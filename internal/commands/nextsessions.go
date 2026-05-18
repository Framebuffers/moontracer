package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*

Flow:
 1. Player types /nextsessions.
 2. Bot loads all upcoming sessions the player is a member of, across all approved
    campaigns, sorted ascending by scheduled time.
 3. Responds ephemerally with a formatted list in the player's timezone.
    If none are found, shows NextSessionsNone.
*/

/*
nextSessionsCommand shows the invoking player's upcoming sessions directly via slash command.

The same data is also reachable via /me -> Next Sessions button (interactions/next_sessions.go).
This command provides a direct entry point without navigating the hub first.
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

	sessions, err := models.GetAllUpcomingSessionsForPlayer(c.db, userID)
	if err != nil {
		log.Printf("nextsessions: load sessions for %s: %v", userID, err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	if len(sessions) == 0 {
		respond(s, i, messages.NextSessionsNone)
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(c.db, userID)
	if err != nil {
		log.Printf("nextsessions: load settings for %s: %v", userID, err)
	}
	loc := time.UTC
	if settings != nil {
		loc = settings.Location()
	}

	var lines []string
	for _, sess := range sessions {
		campaignName := ""
		if sess.Campaign != nil {
			campaignName = sess.Campaign.Name
		}
		formatted := helpers.FormatInLocation(sess.ScheduledAt, messages.SessionListFormat, loc) + " " + helpers.TZLabel(loc)
		lines = append(lines, fmt.Sprintf("• **%s** - %s · %s", campaignName, formatted, helpers.TimeRemaining(sess.ScheduledAt)))
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
