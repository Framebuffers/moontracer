package interactions

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/auth"
	"moontracer/internal/db"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/interactions/router"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

func renderManageSubAuth(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) (*models.Campaign, bool) {
	userID := helpers.GetUserID(i)
	campaign, err := db.GetByID[models.Campaign](database, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return nil, false
	}
	ok, err := auth.Authorize(database, userID, auth.ScopeDM, campaignID)
	if err != nil {
		log.Printf("manage_submenu: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return nil, false
	}
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.ManageNotAuthorized)
		return nil, false
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return nil, false
	}
	return campaign, true
}

// RenderManagePlayersMenu renders the Players sub-menu (Ban, Invite).
func RenderManagePlayersMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, ok := renderManageSubAuth(s, i, database, campaignID)
	if !ok {
		return
	}
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ManageBanLabel,
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("manage_ban:%s", campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageInviteLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageInvitePrefix, campaignID),
			},
		}},
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

// RenderManageSessionsMenu renders the Sessions sub-menu (Set/Reschedule, Announce).
func RenderManageSessionsMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, ok := renderManageSubAuth(s, i, database, campaignID)
	if !ok {
		return
	}
	sessionLabel := messages.ManageSetSessionLabel
	if !campaign.Schedule.NextSession.IsZero() {
		sessionLabel = messages.ManageRescheduleSessionLabel
	}
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    sessionLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageSetSessionPrefix, campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageAnnounceLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("manage_announce:%s", campaignID),
			},
		}},
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

// RenderManageSettingsMenu renders the Settings sub-menu (Set Cover, Set Role, Spicy Zone).
func RenderManageSettingsMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, ok := renderManageSubAuth(s, i, database, campaignID)
	if !ok {
		return
	}
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.SetCoverButtonLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("manage_setcover:%s", campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageSetRoleLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageSetRolePrefix, campaignID),
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			router.NavButton(messages.ManageDangerLabel, discordgo.DangerButton, router.ViewManageDanger, campaignID),
		}},
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

// RenderManageDangerMenu renders the Spicy Zone sub-menu (Archive, Delete).
func RenderManageDangerMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, ok := renderManageSubAuth(s, i, database, campaignID)
	if !ok {
		return
	}
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ManageArchiveLabel,
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageArchivePrefix, campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageDeleteLabel,
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("manage_delete:%s", campaignID),
			},
		}},
		helpers.BackRow(router.ViewManageSettings, campaignID),
	})
}
