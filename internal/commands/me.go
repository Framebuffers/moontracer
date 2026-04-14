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
	Flow:
		1. User runs `/me`.
		2. Authorize: check if registered.
		3. Show player hub with scope-composed action buttons:
		   - Always: My Campaigns, New Campaign, Browse, Next Sessions, Notifications.
		   - If DM of any campaign: + Manage Campaigns.
		   - If Mod/Admin: + Admin Panel.
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

	var row2 []discordgo.MessageComponent
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
