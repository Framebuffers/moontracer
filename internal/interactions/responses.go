package interactions

import (
	"moontracer/internal/interactions/router"
	"moontracer/internal/messages"

	"github.com/bwmarrin/discordgo"
)

/*
Discord interaction response cheat sheet for Moontracer (specifics to bot)

Every interaction must be acknowledged within 3 seconds (a respond* call counts
as acknowledgement). Failing to acknowledge shows the user "this interaction
failed". Calling InteractionRespond twice on the same interaction silently
errors with "interaction has already been acknowledged".

Helpers in this package:

  respondInteraction(s, i, content)
      New ephemeral text. Use for plain success/error messages.
      Wraps InteractionResponseChannelMessageWithSource + ephemeral flag.

  respondUpdate(s, i, content, embeds, components)   [navigation.go]
      Replaces the current ephemeral message in place. Use after a button
      click that should rewrite the same panel (back-nav, menu refresh).
      Wraps InteractionResponseUpdateMessage + ephemeral flag.

  respondWithBackButton(s, i, typ, content, components, view)
      Wraps respondInteraction OR respondUpdate (decided by typ) and appends
      a back-button row that navigates to view.

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
  - Calling respondInteraction twice (second call silently dropped).
  - Forgetting MessageFlagsEphemeral on admin/mod/DM views (leaks to channel).
  - Putting a Button directly in Data.Components without ActionsRow wrapper.
  - Forgetting to clear Embeds on Update (stale embed sticks around).
  - Using strconv/string limits without truncating final content (2000 max).
*/

/*
respondInteraction sends an ephemeral text response to an interaction.
Used by every handler in this package. It is kept here for discoverability.
*/
func respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
respondWithBackButton sends a response that ends with a back-button row
navigating to view. typ controls whether this is a new ephemeral message
(InteractionResponseChannelMessageWithSource) or an in-place update
(InteractionResponseUpdateMessage).

components is a slice of ActionsRows the caller wants above the back-button
row; pass nil if there are none.
*/
func respondWithBackButton(s *discordgo.Session,
	i *discordgo.InteractionCreate,
	typ discordgo.InteractionResponseType,
	content string,
	components []discordgo.MessageComponent,
	view router.ViewID) {
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			router.NavButton(messages.BackLabel, discordgo.PrimaryButton, view),
		},
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: typ,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}
