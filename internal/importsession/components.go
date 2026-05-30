package importsession

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/messages"
)

// BuildStep1Components returns the three select menus (welcome/announcements/sessions) plus nav buttons.
func BuildStep1Components(sessionID string, threads []ThreadOption, sess *Session) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		threadSelectRow(sessionID, "welcome", messages.ImportSelWelcome, threads, currentMapping(sess, "welcome")),
		threadSelectRow(sessionID, "announcements", messages.ImportSelAnnouncements, threads, currentMapping(sess, "announcements")),
		threadSelectRow(sessionID, "sessions", messages.ImportSelSessions, threads, currentMapping(sess, "sessions")),
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ImportCancelLabel,
				Style:    discordgo.DangerButton,
				CustomID: messages.ImportCancelPrefix + ":" + sessionID,
			},
			discordgo.Button{
				Label:    messages.ImportNextLabel,
				Style:    discordgo.PrimaryButton,
				CustomID: messages.ImportNextPrefix + ":" + sessionID,
			},
		}},
	}
}

// BuildStep2Components returns the dice-rolls select menu plus nav buttons.
func BuildStep2Components(sessionID string, threads []ThreadOption, sess *Session) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		threadSelectRow(sessionID, "dice-rolls", messages.ImportSelDiceRolls, threads, currentMapping(sess, "dice-rolls")),
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ImportBackLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: messages.ImportBackPrefix + ":" + sessionID,
			},
			discordgo.Button{
				Label:    messages.ImportCancelLabel,
				Style:    discordgo.DangerButton,
				CustomID: messages.ImportCancelPrefix + ":" + sessionID,
			},
			discordgo.Button{
				Label:    messages.ImportConfirmLabel,
				Style:    discordgo.SuccessButton,
				CustomID: messages.ImportConfirmPrefix + ":" + sessionID,
			},
		}},
	}
}

/*
threadSelectRow builds an ActionsRow with a single select menu for one thread slot.

current is the session's current mapping for this thread (pre-selects the matching option).
*/
func threadSelectRow(sessionID, threadName, placeholder string, threads []ThreadOption, current string) discordgo.ActionsRow {
	opts := []discordgo.SelectMenuOption{
		{
			Label:       messages.ImportOptCreateNew,
			Description: messages.ImportOptCreateNewDescr,
			Value:       messages.ImportCreateNew,
			Default:     current == messages.ImportCreateNew,
		},
	}
	// Note: Discord limits select menus to 25 options max.
	for _, t := range threads {
		if len(opts) >= 25 {
			break
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:   t.Name,
			Value:   t.ID,
			Default: current == t.ID,
		})
	}
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    fmt.Sprintf("%s:%s:%s", messages.ImportThreadSelPrefix, sessionID, threadName),
				Placeholder: placeholder,
				Options:     opts,
			},
		},
	}
}

// currentMapping returns the session's value for threadName, or ImportCreateNew when the session is nil.
func currentMapping(sess *Session, threadName string) string {
	if sess == nil {
		return messages.ImportCreateNew
	}
	return sess.GetCurrentThreadName(threadName)
}
