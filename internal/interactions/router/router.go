package router

/*
	View router.

	Every navigable screen in the bot is identified by a ViewID. Buttons that
	navigate to a view use the CustomID format:

		nav:<view>[:arg1[:arg2…]]

	Flow (click-to-render):
		1. User clicks a button whose CustomID starts with "nav:".
		2. discordgo's prefix dispatcher routes the interaction to navHandler
		   (registered once in registry.go, defined in nav_handler.go).
		3. navHandler calls router.ParseCustomID -> (ViewID, args, ok).
		4. router.Navigate looks up the ViewID in the registry map and invokes
		   the registered RenderFunc with (session, interaction, args).
		5. The RenderFunc (defined in interactions/views.go as a thin adapter)
		   calls the real render function: e.g. commands.RenderMeHub; which
		   issues an InteractionResponseUpdateMessage to replace the current
		   ephemeral message with the new view.

	Flow (startup registration):
		1. main() builds ComponentHandlers via interactions.AllComponents.
		2. AllComponents calls RegisterAllViews(db) before returning.
		3. RegisterAllViews performs a router.Register(ViewID, RenderFunc)
		   call per view, populating the registry map used above.

	Design:
		Back-only. The "parent view" is encoded into each Back button's
		CustomID at render time, so navigation is stateless and survives bot
		restarts. There is no forward button and no in-memory stack: if a user
		wants to go forward, they click the target button again.
*/

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ViewID identifies a renderable view.
type ViewID string

// NavPrefix is the CustomID prefix handled by the nav router.
const NavPrefix = "nav"

// View identifiers. Keep stable: they appear in persisted CustomIDs on live messages.
const (
	ViewMe              ViewID = "me"
	ViewMeCampaigns     ViewID = "me_campaigns"
	ViewMeConfig        ViewID = "me_config"
	ViewMyCampaigns     ViewID = "mycampaigns"
	ViewManage          ViewID = "manage"
	ViewManageCampaign  ViewID = "manage_campaign"
	ViewCampaignsBrowse ViewID = "campaigns_browse"
	ViewCampaignDetail  ViewID = "campaign_detail"
	ViewAdmin           ViewID = "admin"
	ViewAdminDiag       ViewID = "admin_diag"
	ViewAdminDatabase   ViewID = "admin_database"
	ViewAdminCampaigns  ViewID = "admin_campaigns"
	ViewAdminSettings   ViewID = "admin_settings"
	ViewAdminBroadcast  ViewID = "admin_broadcast"
	ViewInbox           ViewID = "inbox"
	ViewConfig          ViewID = "config"
	ViewNextSessions    ViewID = "next_sessions"
	ViewHelp            ViewID = "help"
)

/*
RenderFunc renders a view.

args are positional extras parsed from the CustomID after the ViewID
(e.g. a campaignID for ViewManageCampaign).
*/
type RenderFunc func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string)

var registry = map[ViewID]RenderFunc{}

/*
Register attaches a render function to a ViewID.

Safe to call once at startup; later calls overwrite.
*/
func Register(id ViewID, fn RenderFunc) {
	registry[id] = fn
}

/*
Navigate dispatches to the render function for the given view.

If no render is registered, it logs and responds with an ephemeral error,
so the user isn't left hanging on an interaction timeout.
*/
func Navigate(s *discordgo.Session, i *discordgo.InteractionCreate, id ViewID, args []string) {
	fn, ok := registry[id]
	if !ok {
		log.Printf("router: no renderer for view %q", id)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Navigation error: unknown view %q.", id),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	fn(s, i, args)
}

/*
ParseCustomID splits a "nav:<view>[:arg…]" CustomID into view + args.

Returns ok=false if the string doesn't have the nav prefix.
*/
func ParseCustomID(customID string) (ViewID, []string, bool) {
	parts := strings.Split(customID, ":")
	if len(parts) < 2 || parts[0] != NavPrefix {
		return "", nil, false
	}
	return ViewID(parts[1]), parts[2:], true
}
