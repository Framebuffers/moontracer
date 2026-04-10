package interactions

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
)

// backMe handles back_me: re-renders the /me hub.
type backMe struct {
	db *bun.DB
}

func (h *backMe) CustomIDPrefix() string {
	return "back_me"
}

func (h *backMe) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	commands.RenderMeHub(s, i, userID)
}

// backMyCampaigns handles back_mycampaigns: re-renders the campaign select menu.
type backMyCampaigns struct {
	db *bun.DB
}

func (h *backMyCampaigns) CustomIDPrefix() string {
	return "back_mycampaigns"
}

func (h *backMyCampaigns) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)

	entries, err := models.GetPlayerCampaigns(h.db, userID)
	if err != nil {
		respondInteraction(s, i, messages.MyCampaignsLoadError)
		return
	}

	if len(entries) == 0 {
		respondUpdate(s, i, messages.NoCampaignsMessage, nil, nil)
		return
	}

	selectMenu := BuildPlayerCampaignSelect(entries, messages.MyCampaignSelectPrefix, "Select a campaign...")

	var lines []string
	for _, e := range entries {
		if e.Campaign != nil && e.Campaign.IsApproved {
			lines = append(lines, fmt.Sprintf("**%s** — %s (%s)", e.Campaign.Name, e.Role, e.Status))
		}
	}
	content := "Your campaigns:\n" + strings.Join(lines, "\n")

	respondUpdate(s, i, content, nil, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			backButton(messages.BackLabel, messages.BackMeID),
		}},
	})
}

// backManage handles back_manage: re-renders the /managecampaigns select menu.
type backManage struct {
	db *bun.DB
}

func (h *backManage) CustomIDPrefix() string {
	return "back_manage"
}

func (h *backManage) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)

	entries, err := models.GetPlayerCampaigns(h.db, userID)
	if err != nil {
		respondInteraction(s, i, messages.GenericErrorMessage)
		return
	}

	var dmEntries []models.CampaignPlayer
	for _, e := range entries {
		if e.Role == models.RoleDM {
			dmEntries = append(dmEntries, e)
		}
	}

	if len(dmEntries) == 0 {
		respondUpdate(s, i, messages.ManageNoDMCampaigns, nil, nil)
		return
	}

	selectMenu := BuildPlayerCampaignSelect(dmEntries, messages.ManageSelectPrefix, "Select a campaign to manage...")

	var lines []string
	for _, e := range dmEntries {
		if e.Campaign != nil {
			lines = append(lines, fmt.Sprintf("**%s** — %s", e.Campaign.Name, e.Status))
		}
	}
	content := "Your campaigns (DM):\n" + strings.Join(lines, "\n")

	respondUpdate(s, i, content, nil, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{selectMenu}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			backButton(messages.BackLabel, messages.BackMeID),
			discordgo.Button{
				Label:    messages.NewCampaignLabel,
				Style:    discordgo.SuccessButton,
				CustomID: messages.ManageNewCampaignPrefix,
			},
		}},
	})
}

// backCampaigns handles back_campaigns: re-renders the /campaigns browse view.
type backCampaigns struct {
	db *bun.DB
}

func (h *backCampaigns) CustomIDPrefix() string {
	return "back_campaigns"
}

func (h *backCampaigns) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	RenderCampaignsBrowse(s, i, h.db, "all")
}

// backAdmin handles back_admin: re-renders the /admin hub.
type backAdmin struct {
	db *bun.DB
}

func (h *backAdmin) CustomIDPrefix() string {
	return "back_admin"
}

func (h *backAdmin) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands.RenderAdminHubUpdate(s, i)
}

// backManageCampaign handles back_manage_campaign:<campaignID>: re-renders the manage menu for a specific campaign.
type backManageCampaign struct {
	db *bun.DB
}

func (h *backManageCampaign) CustomIDPrefix() string {
	return "back_manage_campaign"
}

func (h *backManageCampaign) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.SplitN(i.MessageComponentData().CustomID, ":", 2)
	if len(parts) < 2 {
		respondInteraction(s, i, messages.InvalidButtonDataMessage)
		return
	}
	// Delegate to the manage_campaign handler logic
	RenderManageCampaignMenu(s, i, h.db, parts[1])
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}
