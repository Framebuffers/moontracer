package interactions

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/importsession"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

// dmLockedThreads are locked after creation/binding so only Manage Threads holders (the DM) can post.
var dmLockedThreads = map[string]bool{
	"welcome":       true,
	"announcements": true,
}

/*
	Select handler
*/

/*
importThreadSelHandler records one thread mapping choice into the session.

CustomID: import_thread_sel:<sessionID>:<threadName>
*/
type importThreadSelHandler struct{}

func (h *importThreadSelHandler) CustomIDPrefix() string { return messages.ImportThreadSelPrefix }

func (h *importThreadSelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	sessionID := parts[1]
	threadName := parts[2]

	sess, ok := importsession.Get(sessionID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.ImportCampaignErrSession)
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}
	sess.SetThreadMapping(threadName, values[0])

	/*
		Note:
			On an earlier version, I had an issue in the /newcampaign flow where,
			if I had more than one menu, I would choose something on the first menu and then
			delete it when I chose something on the next.

			At the same time, Discord will mark the command as failed if the interaction
			does not respond before 3 secs.

			This saves the chosen option client-side, as to acknowledge that the option
			has been chosen, not change anything visible (as to not make the user think their
			option has been reset), and sets the response as a deferred one, giving the bot
			more time to answer back.
	*/
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

/*
	Next/back handlers
*/

/*
	Cancel handlers
*/

/*
importCancelHandler cancels and discards the session.

CustomID: import_cancel:<sessionID>
*/
type importCancelHandler struct{}

func (h *importCancelHandler) CustomIDPrefix() string { return messages.ImportCancelPrefix }

func (h *importCancelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	importsession.Delete(parts[1])
	helpers.RespondUpdateTerminal(s, i, messages.ImportCampaignCancelled)
}

/*
	Confirm handlers
*/

/*
importConfirmHandler executes the full command with mappings accumulated within this session.

CustomID: import_confirm:<sessionID>
*/
type importConfirmHandler struct {
	db *bun.DB
}

func (h *importConfirmHandler) CustomIDPrefix() string { return messages.ImportConfirmPrefix }

func (h *importConfirmHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 2)
	if !ok {
		return
	}
	sessionID := parts[1]

	sess, ok := importsession.Get(sessionID)
	if !ok {
		helpers.RespondUpdateTerminal(s, i, messages.ImportCampaignErrSession)
		return
	}
	importsession.Delete(sessionID)

	/*
		Remember the 3-second Discord rule: acknowledge immediately and defer the update.

		These kinds of requests usually need a pretty long goroutine to be run, so it's
		a good idea to defer the response for later.
	*/
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	go func() {
		empty := []discordgo.MessageComponent{}
		edit := func(content string) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    &content,
				Components: &empty,
			})
		}
		editWithComponents := func(content string, comps []discordgo.MessageComponent) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    &content,
				Components: &comps,
			})
		}

		/*
			Collect all guild members who hold the campaign role.

			Note:
				Some names *may* have special Unicode characters, because some DMs like to
				*spice up* their tags.

				Normalize this *at some point* as to not break the DB or the character parsing.
				Go should take care of it tho.
		*/
		memberIDs := collectMembersWithRole(s, sess.GuildID, sess.RoleID)
		if !containsString(memberIDs, sess.DMID) {
			memberIDs = append(memberIDs, sess.DMID)
		}

		if err := models.BulkUpsertPlayers(h.db, memberIDs); err != nil {
			log.Printf("importcampaign confirm: bulk upsert players: %v", err)
		}

		tag, err := models.UniqueTag(h.db, models.NormalizeTag(sess.ChannelName))
		if err != nil {
			log.Printf("importcampaign confirm: generate tag for %q: %v", sess.ChannelName, err)
			edit(messages.ImportCampaignErrDB)
			return
		}

		campaign := &models.Campaign{
			ID:            uuid.NewString(),
			Name:          sess.ChannelName,
			Tag:           tag,
			DungeonMaster: sess.DMID,
			ChannelID:     sess.ChannelID,
			RoleID:        sess.RoleID,
			IsApproved:    true,
			IsOpen:        true,
		}
		if _, err := h.db.NewInsert().Model(campaign).Exec(context.Background()); err != nil {
			log.Printf("importcampaign confirm: insert campaign: %v", err)
			edit(messages.ImportCampaignErrDB)
			return
		}

		if err := guard.ChannelPermissionSet(s, sess.ChannelID, s.State.User.ID,
			discordgo.PermissionOverwriteTypeMember,
			discordgo.PermissionViewChannel|discordgo.PermissionManageThreads|discordgo.PermissionSendMessages|discordgo.PermissionManageMessages,
			0,
		); err != nil {
			log.Printf("importcampaign confirm: set bot perms on %s: %v", sess.ChannelID, err)
		}

		created, bound := applyThreadMappings(s, sess, campaign)

		if campaign.AnnouncementsThreadID != "" {
			if _, err := h.db.NewUpdate().Model(campaign).
				Column("announcements_thread_id").
				Where("id = ?", campaign.ID).
				Exec(context.Background()); err != nil {
				log.Printf("importcampaign confirm: save announcements thread id: %v", err)
			}
		}

		if err := models.BulkAddCampaignMembers(h.db, campaign.ID, memberIDs); err != nil {
			log.Printf("importcampaign confirm: bulk add members: %v", err)
		}

		log.Printf("importcampaign: imported %q (%s): %d members, %d bound, %d created",
			sess.ChannelName, campaign.ID, len(memberIDs), bound, created)

		// Try to find an existing billboard forum channel for this campaign's format.
		// If found, post immediately. If not, show a channel selector so the admin can pick one.
		categoryID, catErr := helpers.FindOrCreateCampaignsCategory(s, sess.GuildID)
		if catErr == nil {
			if chanID, ok := helpers.FindForumChannel(s, sess.GuildID, categoryID, billboardChannelName(campaign)); ok {
				if err := PostBillboardToChannel(h.db, s, campaign, chanID); err != nil {
					log.Printf("importcampaign: billboard for %s: %v", campaign.ID, err)
				}
				edit(fmt.Sprintf(messages.ImportCampaignSuccess, sess.ChannelName, len(memberIDs), bound, created))
				return
			}
		}

		// No billboard channel found -ask the admin to select or auto-create one.
		successMsg := fmt.Sprintf(messages.ImportCampaignSuccess, sess.ChannelName, len(memberIDs), bound, created)
		prompt := successMsg + "\n\n" + messages.ImportBillboardPrompt
		editWithComponents(prompt, importBillboardStep3Components(campaign.ID, sess.GuildID))
	}()
}

/*
importCoreThreads are the only threads created/mapped during campaign import.

Social and resources threads are excluded, the DM can create them manually.
*/
var importCoreThreads = []string{"welcome", "announcements", "sessions", "dice-rolls"}

/*
applyThreadMappings binds or creates each core thread based on the user's selections.
It updates campaign.AnnouncementsThreadID in place.
*/
func applyThreadMappings(s *discordgo.Session, sess *importsession.Session, campaign *models.Campaign) (created, bound int) {
	const archiveDuration = 10080 // 1 week in mins

	for _, name := range importCoreThreads {
		threadName := fmt.Sprintf("%s-%s", sess.ChannelName, name)
		choice := sess.GetCurrentThreadName(name)

		var threadID string

		if choice == messages.ImportCreateNew {
			thread, err := guard.ThreadCreate(s, sess.ChannelID, threadName, archiveDuration)
			if err != nil {
				log.Printf("importcampaign: create thread %s: %v", threadName, err)
				continue
			}
			created++
			threadID = thread.ID

			if name == "welcome" {
				initMsg := fmt.Sprintf(messages.ThreadInitMsgWelcomeFmt, campaign.Name)
				msg, err := guard.ChannelMessageSend(s, threadID, initMsg)
				if err != nil {
					log.Printf("importcampaign: init message for %s: %v", threadName, err)
				} else if err := guard.ChannelMessagePin(s, threadID, msg.ID); err != nil {
					log.Printf("importcampaign: pin in %s: %v", threadName, err)
				}
			} else if initMsg, ok := threadInitMessages[name]; ok {
				msg, err := guard.ChannelMessageSend(s, threadID, initMsg)
				if err != nil {
					log.Printf("importcampaign: init message for %s: %v", threadName, err)
				} else if err := guard.ChannelMessagePin(s, threadID, msg.ID); err != nil {
					log.Printf("importcampaign: pin in %s: %v", threadName, err)
				}
			}
		} else {
			threadID = choice
			bound++
		}

		if name == "announcements" {
			campaign.AnnouncementsThreadID = threadID
		}

		if dmLockedThreads[name] {
			if err := guard.LockThread(s, threadID); err != nil {
				log.Printf("importcampaign: lock %s: %v", threadName, err)
			}
		}
	}
	return
}

// collectMembersWithRole paginates GuildMembers and returns IDs of members who hold roleID.
func collectMembersWithRole(s *discordgo.Session, guildID, roleID string) []string {
	var ids []string
	after := ""
	for {
		members, err := s.GuildMembers(guildID, after, 1000)
		if err != nil {
			log.Printf("importcampaign: fetch members (after=%s): %v", after, err)
			break
		}
		for _, m := range members {
			for _, r := range m.Roles {
				if r == roleID {
					ids = append(ids, m.User.ID)
					break
				}
			}
		}
		if len(members) < 1000 {
			break
		}
		after = members[len(members)-1].User.ID
	}
	return ids
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// importBillboardStep3Components builds the channel-select + auto-create row for Step 3.
func importBillboardStep3Components(campaignID, guildID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				MenuType:    discordgo.ChannelSelectMenu,
				CustomID:    fmt.Sprintf("%s:%s:%s", messages.ImportBillboardSelPrefix, campaignID, guildID),
				Placeholder: messages.ImportBillboardSelPlaceholder,
			},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    messages.ImportBillboardSkipLabel,
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("%s:%s:%s", messages.ImportBillboardSkipPrefix, campaignID, guildID),
			},
		}},
	}
}

/*
importBillboardSelHandler handles the admin picking a specific forum channel for the billboard.

CustomID: import_billboard_sel:<campaignID>:<guildID>
*/
type importBillboardSelHandler struct {
	db *bun.DB
}

func (h *importBillboardSelHandler) CustomIDPrefix() string { return messages.ImportBillboardSelPrefix }

func (h *importBillboardSelHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[1]

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	channelID := values[0]

	campaign, ok := helpers.LoadCampaignAsMod(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if err := PostBillboardToChannel(h.db, s, campaign, channelID); err != nil {
		log.Printf("import_billboard_sel: billboard for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ImportCampaignSuccess, campaign.Name, 0, 0, 0))
}

/*
importBillboardSkipHandler auto-creates the billboard forum channel for this campaign's format.

CustomID: import_billboard_skip:<campaignID>:<guildID>
*/
type importBillboardSkipHandler struct {
	db *bun.DB
}

func (h *importBillboardSkipHandler) CustomIDPrefix() string {
	return messages.ImportBillboardSkipPrefix
}

func (h *importBillboardSkipHandler) HandleComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts, ok := helpers.SplitCustomID(s, i, i.MessageComponentData().CustomID, 3)
	if !ok {
		return
	}
	campaignID := parts[1]
	guildID := parts[2]

	campaign, ok := helpers.LoadCampaignAsMod(s, i, h.db, campaignID)
	if !ok {
		return
	}

	if err := PostBillboard(h.db, s, campaign, guildID); err != nil {
		log.Printf("import_billboard_skip: billboard for %s: %v", campaignID, err)
		helpers.RespondUpdateTerminal(s, i, messages.GenericErrorMessage)
		return
	}
	helpers.RespondUpdateTerminal(s, i, fmt.Sprintf(messages.ImportCampaignSuccess, campaign.Name, 0, 0, 0))
}
