package interactions

/*
	View registration.

	RegisterAllViews is called once at startup (from registry.go) and wires every
	ViewID in the router to its render function. Adapters normalize render signatures
	into the router's RenderFunc shape (s, i, args).
*/

import (
	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/interactions/router"
)

/*
RegisterAllViews wires every router ViewID to a package-local render function.

Call once at startup before the bot starts handling interactions.
*/
func RegisterAllViews(db *bun.DB) {
	router.Register(router.ViewMe, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		commands.RenderMeHub(s, i, db, getUserID(i))
	})

	router.Register(router.ViewMyCampaigns, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderMyCampaignsList(s, i, db, getUserID(i))
	})

	router.Register(router.ViewManage, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		RenderManageList(s, i, db, getUserID(i))
	})

	router.Register(router.ViewManageCampaign, func(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) {
		if len(args) < 1 {
			return
		}
		RenderManageCampaignMenu(s, i, db, args[0])
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
		commands.RenderAdminDiag(s, i)
	})
}
