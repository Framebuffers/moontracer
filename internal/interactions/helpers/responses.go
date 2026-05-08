package helpers

import (
	"github.com/bwmarrin/discordgo"

	"moontracer/internal/interactions/router"
	"moontracer/internal/messages"
)

/*
Discord interaction response cheat sheet for Moontracer (specifics to bot)

Every interaction must be acknowledged within 3 seconds (a respond* call counts
as acknowledgement). Failing to acknowledge shows the user "this interaction
failed". Calling InteractionRespond twice on the same interaction silently
errors with "interaction has already been acknowledged".

Helpers in this package:

  Respond(s, i, content)
      New ephemeral text. Use for plain success/error messages.
      Wraps InteractionResponseChannelMessageWithSource + ephemeral flag.

  RespondUpdate(s, i, content, embeds, components)
      Replaces the current ephemeral message in place. Use after a button
      click that should rewrite the same panel (back-nav, menu refresh).
      Wraps InteractionResponseUpdateMessage + ephemeral flag.

  RespondWithBack(s, i, typ, content, components, view)
      Wraps Respond OR RespondUpdate (decided by typ) and appends a
      back-button row that navigates to view.

Components rules:

  - Top-level Data.Components entries MUST be ActionsRow (or V2 containers).
    A bare Button at the top level is rejected by Discord.
  - 5 ActionsRows max per message. 5 components per row, OR 1 SelectMenu alone.
  - Buttons need either a Label or Emoji, plus a CustomID (non-link buttons).

Embeds:

  - On InteractionResponseUpdateMessage, set Embeds: []*MessageEmbed{} to drop
    the previous embeds. Leaving it nil keeps them in place.

Modals:

  - InteractionResponseModal uses Data.CustomID + Title + Components
    (TextInputs wrapped in ActionsRows). Modals can NOT be ephemeral.
  - The submit fires a separate interaction; read i.ModalSubmitData().

Common mistakes:

  - Returning from a handler without responding (interaction hangs).
  - Calling Respond twice (second call silently dropped).
  - Forgetting MessageFlagsEphemeral on admin/mod/DM views (leaks to channel).
  - Putting a Button directly in Data.Components without ActionsRow wrapper.
  - Forgetting to clear Embeds on Update (stale embed sticks around).
  - Truncating per-line instead of final content (2000 char limit applies to
    the assembled Content, not each line).
*/

/*
Respond sends an ephemeral text response to an interaction.
Used by every handler in the interactions package.
*/
func Respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
RespondUpdate replaces the current message instead of sending a new one.
Used by back buttons and select menus to update in place.
*/
func RespondUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Embeds:     embeds,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
RespondUpdateTerminal updates the current message with a terminal text response
and a Home button. Use in component handlers for final success and error states
that end the current flow.
*/
func RespondUpdateTerminal(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	RespondUpdate(s, i, content, []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
		}},
	})
}

/*
BackRow builds an ActionsRow with a back button pointing at target.
When target is not ViewMe, a secondary Home button (-> ViewMe) is appended so
users can always return to the player hub from any depth.
*/
func BackRow(target router.ViewID, args ...string) discordgo.ActionsRow {
	if target == router.ViewMe {
		// Back and Home are the same destination here; one button avoids a duplicate CustomID.
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
		}}
	}
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.BackButton(messages.BackLabel, target, args...),
		router.NavButton(messages.HomeLabel, discordgo.SecondaryButton, router.ViewMe),
	}}
}

/*
RespondWithBack sends a response that ends with a back-button row navigating to
view. typ controls whether this is a new ephemeral message
(InteractionResponseChannelMessageWithSource) or an in-place update
(InteractionResponseUpdateMessage).

components is a slice of ActionsRows the caller wants above the back-button
row; pass nil if there are none.
*/
func RespondWithBack(s *discordgo.Session,
	i *discordgo.InteractionCreate,
	typ discordgo.InteractionResponseType,
	content string,
	components []discordgo.MessageComponent,
	view router.ViewID) {
	components = append(components, BackRow(view))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: typ,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
