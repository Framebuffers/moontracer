package interactions

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
Flow:
 1. Button (manage_unban:<campaignID>): loads StatusBanned players, shows select menu.
    Empty state: ManageUnbanNoBanned.
 2. Select (manage_unban_select): value format "campaignID:playerID".
    Sets StatusFinished (not Active — DM expelled them; reinvite is a separate action).
    Restores the campaign Discord role so they can see the channel and press Join.
    DMs the player to let them know.
*/

type manageCampaignUnban struct {
	db *bun.DB
}

func (h *manageCampaignUnban) CustomIDPrefix() string {
	return messages.ManageUnbanPrefix
}

func (h *manageCampaignUnban) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := renderManageSubAuth(s, i, h.db, campaignID)
	if !ok {
		return
	}

	players, err := models.GetCampaignPlayers(h.db, campaignID)
	if err != nil {
		log.Printf("manage_unban: failed to load players for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	var opts []discordgo.SelectMenuOption
	for _, p := range players {
		if p.Status != models.StatusBanned {
			continue
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: helpers.MemberName(s, i.GuildID, p.PlayerID),
			Value: fmt.Sprintf("%s:%s", campaignID, p.PlayerID),
		})
	}

	if len(opts) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.ManageUnbanNoBanned)
		return
	}

	helpers.RespondUpdate(s, i, fmt.Sprintf(messages.ManageCampaignHeader, campaign.Name), []*discordgo.MessageEmbed{}, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    messages.ManageUnbanSelectPrefix,
				Placeholder: messages.ManageUnbanSelectPlaceholder,
				Options:     opts,
			},
		}},
	})
}

type manageCampaignUnbanSelect struct {
	db *bun.DB
}

func (h *manageCampaignUnbanSelect) CustomIDPrefix() string {
	return messages.ManageUnbanSelectPrefix
}

func (h *manageCampaignUnbanSelect) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := helpers.GetUserID(i)

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}

	raw := strings.SplitN(values[0], ":", 2)
	if len(raw) < 2 {
		helpers.RespondUpdateTerminal(s, i, messages.InvalidButtonDataMessage)
		return
	}
	campaignID := raw[0]
	targetID := raw[1]

	authorized, err := auth.Authorize(h.db, invokerID, auth.ScopeDM, campaignID)
	if err != nil {
		log.Printf("manage_unban_select: auth check failed: %v", err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	if !authorized {
		helpers.RespondUpdateTerminal(s, i, messages.ManageNotAuthorized)
		return
	}

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	if err := models.SetCampaignPlayerStatus(h.db, targetID, campaignID, models.StatusFinished); err != nil {
		log.Printf("manage_unban_select: failed to set status for %s in %s: %v", targetID, campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.ManageUnbanFailure)
		return
	}

	if campaign.RoleID != "" {
		if err := guard.GuildMemberRoleAdd(s, i.GuildID, targetID, campaign.RoleID); err != nil {
			log.Printf("manage_unban_select: failed to restore role %s for %s: %v", campaign.RoleID, targetID, err)
		}
	}

	go func() {
		ch, err := s.UserChannelCreate(targetID)
		if err != nil {
			log.Printf("manage_unban_select: failed to open DM channel for %s: %v", targetID, err)
			return
		}
		if _, err := guard.ChannelMessageSend(s, ch.ID, fmt.Sprintf(messages.ManageUnbanDMMessage, campaign.Name)); err != nil {
			log.Printf("manage_unban_select: failed to send unban DM to %s: %v", targetID, err)
		}
	}()

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageUnbanSuccess, targetID, campaign.Name))
}
