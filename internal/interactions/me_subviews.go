package interactions

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

/*
RenderMeCampaigns renders the Campaigns sub-view of the /me hub.

Buttons: My Campaigns (grey) | New Campaign (green) | Browse (grey) | Back -> ViewMe
*/
func RenderMeCampaigns(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: messages.MeCampaignsHeader,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					router.NavButton(messages.MyCampaignsLabel, discordgo.SecondaryButton, router.ViewMyCampaigns),
					discordgo.Button{
						Label:    messages.NewCampaignLabel,
						Style:    discordgo.SuccessButton,
						CustomID: messages.ManageNewCampaignPrefix,
					},
					router.NavButton(messages.BrowseCampaignsLabel, discordgo.SecondaryButton, router.ViewCampaignsBrowse, "all"),
				}},
				helpers.BackRow(router.ViewMe),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

/*
RenderMeConfig renders the Configuration sub-view of the /me hub.

Fixed buttons: Timezone (grey), Notifications (grey).
Conditional: Manage Campaigns (grey) if DM; Admin Panel (red) if mod/admin.
*/
func RenderMeConfig(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB, userID string) {
	row := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.TimezoneLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.TimezonePrefix,
		},
		discordgo.Button{
			Label:    messages.NotificationsLabel,
			Style:    discordgo.SecondaryButton,
			CustomID: messages.NotificationsPrefix,
		},
	}

	if isDMOfAnyCampaign(db, userID) {
		row = append(row, router.NavButton(messages.ControlPanelLabel, discordgo.SecondaryButton, router.ViewManage))
	}
	if isMod, err := auth.Authorize(db, userID, auth.ScopeMod, ""); err == nil && isMod {
		row = append(row, router.NavButton(messages.AdminPanelLabel, discordgo.DangerButton, router.ViewAdmin))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: messages.MeConfigHeader,
			Embeds:  []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: row},
				helpers.BackRow(router.ViewMe),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// isDMOfAnyCampaign returns true if the user has at least one campaign membership with RoleDM.
func isDMOfAnyCampaign(db *bun.DB, userID string) bool {
	entries, err := models.GetPlayerCampaigns(db, userID)
	if err != nil {
		log.Printf("me_subviews: failed to probe DM status for %s: %v", userID, err)
		return false
	}
	for _, e := range entries {
		if e.Role == models.RoleDM {
			return true
		}
	}
	return false
}
