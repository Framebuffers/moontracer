package interactions

/*
Thread remapping for existing campaigns.

An admin or the DM can re-link any of the four core thread slots after an import if they
accidentally chose the wrong thread.

The flow mirrors import step 1 but updates an existing campaign's stored thread IDs
instead of creating a new one.

Flow:
 1. DM clicks "Remap Threads" in manage -> Settings.
 2. manageRemapThreadsHandler fetches all threads in the campaign channel, creates
    an importsession pre-filled with current mappings, shows the step-1 selector.
 3. importThreadSelHandler (already registered) records changes into the session.
 4. DM clicks Confirm -> manageRemapConfirmHandler saves new thread IDs to DB.
*/

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/importsession"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

type manageRemapThreadsHandler struct {
	db *bun.DB
}

func (h *manageRemapThreadsHandler) CustomIDPrefix() string {
	return messages.ManageRemapThreadsPrefix
}

func (h *manageRemapThreadsHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	campaignID := parts[1]

	campaign, ok := helpers.LoadCampaignAsDM(s, i, h.db, campaignID)
	if !ok {
		return
	}
	if !helpers.IsCampaignMutable(s, i, campaign) {
		return
	}
	if !messages.IsSnowflake(campaign.ChannelID) {
		helpers.RespondUpdateTerminal(s, i, "⚠️ This campaign has no linked channel. Import it first.")
		return
	}

	threads := commands.FetchExistingThreads(s, i.GuildID, campaign.ChannelID)

	sessionID, sess := importsession.New(i.GuildID, campaign.ChannelID, campaign.Tag, campaign.RoleID, campaign.DungeonMaster, threads)

	// prefill so slots keep their current value
	if campaign.AnnouncementsThreadID != "" {
		sess.SetThreadMapping("announcements", campaign.AnnouncementsThreadID)
	}

	/*
		Reuse the 4 select-menu rows from BuildStep1Components, swap the nav row for
		the remap-specific confirm button (which encodes campaignID).
	*/
	baseComps := importsession.BuildStep1Components(sessionID, threads, sess)
	comps := append(baseComps[:4], discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label:    messages.ImportCancelLabel,
			Style:    discordgo.DangerButton,
			CustomID: messages.ImportCancelPrefix + ":" + sessionID,
		},
		discordgo.Button{
			Label:    "✅ Save Remapping",
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("%s:%s:%s", messages.ManageRemapConfirmPrefix, sessionID, campaignID),
		},
	}})

	content := fmt.Sprintf(messages.ManageRemapHeader, campaign.Name)
	helpers.RespondUpdate(s, i, content, []*discordgo.MessageEmbed{}, comps)
}

type manageRemapConfirmHandler struct {
	db *bun.DB
}

func (h *manageRemapConfirmHandler) CustomIDPrefix() string {
	return messages.ManageRemapConfirmPrefix
}

func (h *manageRemapConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// CustomID: manage_remap_confirm:<sessionID>:<campaignID>
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	sessionID := parts[1]
	campaignID := parts[2]

	// Allow both DM and mod to remap.
	userID := helpers.GetUserID(i)
	dmOK, _ := auth.Authorize(h.db, userID, auth.ScopeDM, campaignID)
	modOK, _ := auth.Authorize(h.db, userID, auth.ScopeMod, "")
	if !dmOK && !modOK {
		helpers.RespondUpdateTerminal(s, i, messages.ManageNotAuthorized)
		return
	}

	sess, ok := importsession.Get(sessionID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.ImportCampaignErrSession)
		return
	}
	importsession.Delete(sessionID)

	campaign, err := db.GetByID[models.Campaign](h.db, campaignID)
	if err != nil {
		helpers.RespondUpdateTerminal(s, i, messages.ManageCampaignNotFound)
		return
	}

	// Apply each mapped slot. "new" means the user left it unchanged; skip those.
	for _, name := range importCoreThreads {
		choice := sess.GetCurrentThreadName(name)
		if choice == messages.ImportCreateNew {
			continue // unchanged
		}
		switch name {
		case "announcements":
			campaign.AnnouncementsThreadID = choice
		}
	}

	if _, err := h.db.NewUpdate().Model(campaign).
		Column("announcements_thread_id").
		Where("id = ?", campaign.ID).
		Exec(context.Background()); err != nil {
		log.Printf("manage_remap_confirm: save thread IDs for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}

	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ManageRemapSuccess, campaign.Name))
}
