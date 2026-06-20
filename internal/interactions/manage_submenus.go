package interactions

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/interactions/router"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
	Flow:
		Triggered from the top-level manage-campaign menu (a per-campaign hub
		reached from /managecampaigns -> select).

		This file owns the four sub-menus below that hub, plus one action button.

		Each Render* function is invoked by router.ViewManage* nav targets:
		1. RenderManagePlayersMenu (ViewManagePlayers):
			a. Buttons: Ban, Invite, Download Tokens.
			b. [Back -> ViewManageCampaign]
		2. RenderManageSessionsMenu (ViewManageSessions):
			a. Button label flips Set/Reschedule based on Schedule.NextSession.
			b. Buttons: Set/Reschedule Session, Announce.
			c. [Back -> ViewManageCampaign]
		3. RenderManageSettingsMenu (ViewManageSettings):
			a. Buttons: Set Cover, Set Role, Edit Links, Open/Close toggle.
			b. Second row: Spicy Zone (danger nav).
			c. [Back -> ViewManageCampaign]
		4. RenderManageDangerMenu (ViewManageDanger):
			a. Buttons: Archive, Delete.
			b. [Back -> ViewManageSettings]
		5. manageDownloadTokensHandler (manage_download_tokens:<campaignID>):
			a. DM-auth check (auth.ScopeDM).
			b. Loads CampaignPlayers with media_id != '' and their Media relation.
			c. Opens each file from disk and replies with all attached at once,
			   deferred-cleanup via handles slice + defer.
			d. Empty state -> ManageDownloadTokensNone terminal reply.

	Notes:
		- renderManageSubAuth centralizes the load-campaign + DM-auth + mutable
		  triple-check used by every sub-menu render. Returns (campaign, false)
		  and writes the error reply on any failure.
		- Mutable check rejects archived campaigns: archived campaigns can be
		  viewed but not managed.
*/

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
			discordgo.Button{
				Label:    messages.ManageDownloadTokensLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageDownloadTokensPrefix, campaignID),
			},
		}},
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

// RenderManageSessionsMenu renders the Sessions sub-menu (New Session).
func RenderManageSessionsMenu(s *discordgo.Session, i *discordgo.InteractionCreate, database *bun.DB, campaignID string) {
	campaign, ok := renderManageSubAuth(s, i, database, campaignID)
	if !ok {
		return
	}
	_ = campaign // loaded for auth; not needed for the static menu
	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, "Sessions"), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ManageNewSessionLabel,
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageNewSessionPrefix, campaignID),
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
	toggleLabel := messages.ManageCloseLabel
	if !campaign.IsOpen {
		toggleLabel = messages.ManageOpenLabel
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
			discordgo.Button{
				Label:    messages.ManageLinksLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageLinksPrefix, campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageGameInfoLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageGameInfoPrefix, campaignID),
			},
			discordgo.Button{
				Label:    toggleLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("campaign_toggle:%s", campaignID),
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ManageLinkRoleLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageLinkRolePrefix, campaignID),
			},
			discordgo.Button{
				Label:    messages.ManageRemapThreadsLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s", messages.ManageRemapThreadsPrefix, campaignID),
			},
			router.NavButton(messages.ManageDangerLabel, discordgo.DangerButton, router.ViewManageDanger, campaignID),
		}},
		helpers.BackRow(router.ViewManageCampaign, campaignID),
	})
}

/*
manageDownloadTokensHandler sends all player tokens for a campaign as file attachments.

Only the DM can trigger this. Skips players with no token assigned.

CustomID: manage_download_tokens:<campaignID>
*/
type manageDownloadTokensHandler struct {
	db *bun.DB
}

func (h *manageDownloadTokensHandler) CustomIDPrefix() string {
	return messages.ManageDownloadTokensPrefix
}

func (h *manageDownloadTokensHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]
	userID := helpers.GetUserID(i)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}
	if authorized, _ := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID); !authorized {
		helpers.RespondUpdateTerminal(s, i, messages.ManageNotAuthorized)
		return
	}

	var players []models.CampaignPlayer
	if err := h.db.NewSelect().Model(&players).
		Relation("Media").
		Where("campaign_player.campaign_id = ?", campaignID).
		Where("campaign_player.media_id != ''").
		Scan(context.Background()); err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if len(players) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.ManageDownloadTokensNone)
		return
	}

	var handles []*os.File
	defer func() {
		for _, f := range handles {
			f.Close()
		}
	}()

	var files []*discordgo.File
	for _, cp := range players {
		if cp.Media == nil {
			continue
		}
		f, err := os.Open(cp.Media.Path)
		if err != nil {
			log.Printf("manage_download_tokens: open %s: %v", cp.Media.Path, err)
			continue
		}
		handles = append(handles, f)
		name := cp.Media.Name
		if name == "" {
			name = cp.PlayerID[:8]
		}
		files = append(files, &discordgo.File{
			Name:        name + ".png",
			ContentType: "image/png",
			Reader:      f,
		})
	}

	if len(files) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.ManageDownloadTokensNone)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(messages.ManageDownloadTokensContent, campaign.Name, len(files)),
			Files:   files,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
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
