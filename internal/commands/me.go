package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
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
			Components: buildMeHubComponents(m.db, userID),
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
			Components: buildMeHubComponents(db, userID),
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
buildMeHubComponents assembles the /me button layout based on the caller's scope.

Layout:
  - Row 1 (Campaigns): My Campaigns | New Campaign | Browse
  - Row 2 (Activity):  Next Sessions | Notifications
  - Row 3 (Options):   Timezone [| Manage Campaigns (DM only)] [| Admin Panel (mod only)]

Failures on scope probes downgrade gracefully: the button is omitted
rather than blocking the whole hub.
*/
func buildMeHubComponents(db *bun.DB, userID string) []discordgo.MessageComponent {
	rowCampaigns := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		router.NavButton(messages.MyCampaignsLabel, discordgo.PrimaryButton, router.ViewMyCampaigns),
		discordgo.Button{
			Label:    messages.NewCampaignLabel,
			Style:    discordgo.PrimaryButton,
			CustomID: messages.ManageNewCampaignPrefix,
		},
		router.NavButton(messages.BrowseCampaignsLabel, discordgo.SecondaryButton, router.ViewCampaignsBrowse, "all"),
	}}

	rowActivity := discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.NextSessionsLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.NextSessionsPrefix,
		},
		discordgo.Button{
			Label:    messages.NotificationsLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.NotificationsPrefix,
		},
	}}

	rowOptions := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.TimezoneLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.TimezonePrefix,
		},
	}
	if isDMOfAnyCampaign(db, userID) {
		rowOptions = append(rowOptions, router.NavButton(messages.ManageCampaignsLabel, discordgo.PrimaryButton, router.ViewManage))
	}
	if isMod, err := auth.Authorize(db, userID, auth.ScopeMod, ""); err == nil && isMod {
		rowOptions = append(rowOptions, router.NavButton(messages.AdminPanelLabel, discordgo.DangerButton, router.ViewAdmin))
	}

	return []discordgo.MessageComponent{
		rowCampaigns,
		rowActivity,
		discordgo.ActionsRow{Components: rowOptions},
	}
}

// isDMOfAnyCampaign returns true if the user has at least one campaign membership with RoleDM.
func isDMOfAnyCampaign(db *bun.DB, userID string) bool {
	entries, err := models.GetPlayerCampaigns(db, userID)
	if err != nil {
		log.Printf("me: failed to probe DM status for %s: %v", userID, err)
		return false
	}
	for _, e := range entries {
		if e.Role == models.RoleDM {
			return true
		}
	}
	return false
}
