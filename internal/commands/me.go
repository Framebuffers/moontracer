package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/interactions/router"
	"moontracer/internal/messages"
)

/*
	/me is the player hub. The landing page every other player-facing view
	backs out to. Button layout is composed from the invoker's scope so users
	only see affordances they can actually use.

	Flow (slash command):
		1. User runs `/me`.
		2. Authorize ScopePlayer (is the user registered?). Unregistered ->
		   NotRegisteredMessage and abort.
		3. Call buildMeHubComponents(db, userID) to produce a scope-aware
		   []MessageComponent.
		4. Respond with InteractionResponseChannelMessageWithSource
		   (fresh ephemeral message).

	Flow (back navigation → RenderMeHub):
		1. User clicks "Back" on any child view that pointed at ViewMe.
		2. navHandler -> router.Navigate -> the ViewMe adapter in views.go.
		3. Adapter calls RenderMeHub(s, i, db, userID).
		4. RenderMeHub re-runs buildMeHubComponents and responds with
		   InteractionResponseUpdateMessage (replaces the current view
		   in place, no new ephemeral).

	Flow (buildMeHubComponents scope composition):
		1. Row 1 (Campaigns) is fixed: My Campaigns, New Campaign, Browse.
		2. Row 2 (Activity) is fixed: Next Sessions, Notifications.
		3. Row 3 (Options) starts with Timezone (always shown).
		   - Probe isDMOfAnyCampaign: if true, add Manage Campaigns.
		   - Probe auth.Authorize(ScopeMod): if true, add Admin Panel.
		     ScopeAdmin is a superset of ScopeMod so this covers both.
		4. On probe failure (DB error), the button is silently omitted
		   rather than blocking the whole hub — /me must always render
		   something for a registered user.
*/

type meCommand struct {
	db *bun.DB
}

func (m *meCommand) Data() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        messages.MeCommandName,
		Description: messages.MeCommandDesc,
	}
}

func (m *meCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	registered, err := auth.Authorize(m.db, userID, auth.ScopePlayer, "")
	if err != nil {
		log.Printf("me: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !registered {
		respond(s, i, messages.NotRegisteredMessage)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf(messages.MeHubMessage, userID),
			Components: buildMeHubComponents(),
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// RenderMeHub re-renders the /me hub via a message update (used by router back buttons).
func RenderMeHub(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf(messages.MeHubMessage, userID),
			Embeds:     []*discordgo.MessageEmbed{},
			Components: buildMeHubComponents(),
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
buildMeHubComponents returns the three-button /me hub row.

All hub buttons are blurple (Primary). Sub-views handle scoped content.
*/
func buildMeHubComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.MeCampaignsLabel, discordgo.PrimaryButton, router.ViewMeCampaigns),
			discordgo.Button{
				Label:    messages.NextSessionsLabel,
				Style:    discordgo.PrimaryButton,
				CustomID: messages.NextSessionsPrefix,
			},
			router.NavButton(messages.MeConfigLabel, discordgo.PrimaryButton, router.ViewMeConfig),
			router.NavButton(messages.AboutLabel, discordgo.SecondaryButton, router.ViewAbout),
		}},
	}
}

