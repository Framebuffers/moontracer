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
thisWeekCommand lists all sessions from approved campaigns scheduled in the next 7 days.

Players can see what's available to play for that week, so they could join or spectate.
They can see any campaign that's: approved, non-archived and scheduled.
*/
type thisWeekCommand struct {
	db *bun.DB
}

func (c *thisWeekCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.ThisWeekCommandName,
		Description: messages.ThisWeekCommandDesc,
	}
}

func (c *thisWeekCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	sessions, err := models.GetSessionsThisWeek(c.db)
	if err != nil {
		log.Printf("thisweek: load sessions: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}

	if len(sessions) == 0 {
		respond(s, i, messages.ThisWeekNone)
		return
	}

	settings, err := models.GetOrCreatePlayerSettings(c.db, userID)
	if err != nil {
		log.Printf("thisweek: load settings for %s: %v", userID, err)
	}
	loc := time.UTC
	if settings != nil {
		loc = settings.Location()
	}

	var lines []string
	for _, sess := range sessions {
		if sess.Campaign == nil {
			continue
		}
		camp := sess.Campaign

		formatted := helpers.FormatInLocation(sess.ScheduledAt, messages.SessionListFormat, loc) +
			" " + helpers.TZLabel(loc)

		var availability string
		switch {
		case !camp.IsOpen:
			availability = messages.ThisWeekClosed
		case camp.SessionCapacity == 0:
			availability = messages.ThisWeekSlotsOpen
		default:
			accepted, _ := models.CountAcceptedPlayers(c.db, sess.ID)
			availability = fmt.Sprintf(messages.ThisWeekSlotsFmt, camp.SessionCapacity-accepted, camp.SessionCapacity)
		}

		title := camp.Name
		if sess.Title != "" {
			title = fmt.Sprintf("%s — %s", camp.Name, sess.Title)
		}

		lines = append(lines, fmt.Sprintf("• **%s** · %s · %s · _%s_",
			title, formatted, helpers.TimeRemaining(sess.ScheduledAt), availability))
	}

	content := messages.ThisWeekHeader + "\n" + strings.Join(lines, "\n")
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
