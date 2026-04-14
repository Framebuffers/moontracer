package router

/*
	Button constructors for router-driven navigation.

	Use NavButton for forward links (most buttons that jump to another view) and
	BackButton for the conventional "◀ Back" button on any non-hub view.
*/

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

/*
NavCustomID builds a "nav:<view>[:arg1…]" CustomID string.

Useful when you need the raw string (e.g. inside a select-menu option) instead of a full button.
*/
func NavCustomID(target ViewID, args ...string) string {
	parts := make([]string, 0, 2+len(args))
	parts = append(parts, NavPrefix, string(target))
	parts = append(parts, args...)
	return strings.Join(parts, ":")
}

// NavButton builds a clickable button that routes to the given view.
func NavButton(label string, style discordgo.ButtonStyle, target ViewID, args ...string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    style,
		CustomID: NavCustomID(target, args...),
	}
}

/*
BackButton is a secondary-styled button with a back-arrow emoji, pointing to
the given target view.

Label is supplied so callers can localize.
*/
func BackButton(label string, target ViewID, args ...string) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.SecondaryButton,
		CustomID: NavCustomID(target, args...),
		Emoji:    &discordgo.ComponentEmoji{Name: "◀"},
	}
}
