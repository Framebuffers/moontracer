package interactions

/*
	View registration.

	This file is the single place where ViewIDs are bound to render functions.
	The actual rendering bodies live next to the feature they render
	(commands.RenderMeHub, interactions.RenderManageCampaignMenu, etc.):
	RegisterAllViews only glues them to the router.

	Flow (startup):
		1. main() calls interactions.AllComponents(db, dispatcher).
		2. AllComponents calls RegisterAllViews(db) as its first statement,
		   before returning the ComponentHandler list.
		3. For each ViewID, RegisterAllViews calls router.Register with an
		   adapter closure of shape RenderFunc — func(s, i, args).
		4. The adapter unpacks `args` (e.g. args[0] as a campaignID) and
		   forwards to the underlying render function with its natural
		   signature. This keeps render functions free of the router's
		   positional-args convention.
		5. Once this completes, router.Navigate can resolve any registered
		   ViewID for the lifetime of the process.
*/

import (
	"moontracer/internal/interactions/helpers"
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/dispatch"
	"moontracer/internal/interactions/router"
)

/*
RegisterAllViews wires every router ViewID to a package-local render function.

Call once at startup before the bot starts handling interactions.
*/
func RegisterAllViews(db *bun.DB, d *dispatch.Dispatcher) {
	router.Register(router.ViewMe, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		commands.RenderMeHub(s, i, db, helpers.GetUserID(i))
	})

	router.Register(router.ViewMeCampaigns, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderMeCampaigns(s, i, db, helpers.GetUserID(i))
	})

	router.Register(router.ViewMeConfig, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderMeConfig(s, i, db, helpers.GetUserID(i))
	})

	router.Register(router.ViewMyCampaigns, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderMyCampaignsList(s, i, db, helpers.GetUserID(i))
	})

	router.Register(router.ViewManage, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderManageList(s, i, db, helpers.GetUserID(i))
	})

	router.Register(router.ViewManageCampaign, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManageCampaignMenu(s, i, db, args[0])
	})

	router.Register(router.ViewManagePlayers, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManagePlayersMenu(s, i, db, args[0])
	})

	router.Register(router.ViewManageSessions, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManageSessionsMenu(s, i, db, args[0])
	})

	router.Register(router.ViewManageSettings, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManageSettingsMenu(s, i, db, args[0])
	})

	router.Register(router.ViewManageDanger, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManageDangerMenu(s, i, db, args[0])
	})

	router.Register(router.ViewCampaignsBrowse, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		filter := "all"
		if len(args) > 0 && args[0] != "" {
			filter = args[0]
		}
		RenderCampaignsBrowse(s, i, db, filter)
	})

	router.Register(router.ViewCampaignDetail, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderCampaignDetail(s, i, db, args[0])
	})

	router.Register(router.ViewAdmin, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		commands.RenderAdminHubUpdate(s, i)
	})

	router.Register(router.ViewAdminDiag, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		commands.RenderAdminDiag(s, i, db, i.GuildID)
	})

	router.Register(router.ViewHelp, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		commands.RenderHelp(s, i, db, d)
	})
}
