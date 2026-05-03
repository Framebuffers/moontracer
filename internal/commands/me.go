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
		1. Start with a fixed row1: My Campaigns, New Campaign, Browse,
		   Next Sessions, Notifications. These are available to every
		   registered player.
		2. Probe isDMOfAnyCampaign(db, userID): loads the user's
		   CampaignPlayer rows and checks for any Role == DM. If true,
		   append a "Manage Campaigns" button to row2.
		3. Probe auth.Authorize(ScopeMod). If true, append an "Admin Panel"
		   button to row2. ScopeAdmin is a superset of ScopeMod so this
		   check covers both.
		4. On probe failure (DB error), the button is silently omitted
		   rather than blocking the whole hub — /me must always render
		   something for a registered user.
		5. Return [row1] if row2 is empty, else [row1, row2].
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

Failures on scope probes downgrade gracefully: the button is omitted
rather than blocking the whole hub.
*/
func buildMeHubComponents(db *bun.DB, userID string) []discordgo.MessageComponent {
	row1 := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.MyCampaignsLabel,
			Style:    discordgo.PrimaryButton,
			CustomID: router.NavCustomID(router.ViewMyCampaigns),
		},
		discordgo.Button{
			Label:    messages.NewCampaignLabel,
			Style:    discordgo.PrimaryButton,
			CustomID: messages.ManageNewCampaignPrefix,
		},
		discordgo.Button{
			Label:    messages.BrowseCampaignsLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: router.NavCustomID(router.ViewCampaignsBrowse, "all"),
		},
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
	}

	row2 := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.TimezoneLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.TimezonePrefix,
		},
	}
	if isDMOfAnyCampaign(db, userID) {
		row2 = append(row2, discordgo.Button{
			Label:    messages.ManageCampaignsCommandDesc,
			Style:    discordgo.PrimaryButton,
			CustomID: router.NavCustomID(router.ViewManage),
		})
	}
	if isMod, err := auth.Authorize(db, userID, auth.ScopeMod, ""); err == nil && isMod {
		row2 = append(row2, discordgo.Button{
			Label:    messages.AdminPanelLabel,
			Style:    discordgo.DangerButton,
			CustomID: router.NavCustomID(router.ViewAdmin),
		})
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: row1},
	}
	if len(row2) > 0 {
		components = append(components, discordgo.ActionsRow{Components: row2})
	}
	return components
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
