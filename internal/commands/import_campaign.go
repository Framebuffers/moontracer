package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/auth"
	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
Flow:
 1. Staff invokes /importcampaign with an existing channel, role, and DM.
 2. Bot defers an ephemeral response immediately (work can take several seconds).
 3. In a goroutine:
    a. Read channel metadata (name, category) from Discord.
    b. Paginate GuildMembers to collect all members who hold the campaign role.
    c. Bulk-upsert Player rows for every collected member (including the DM),
       preserving any existing role/ban state via ON CONFLICT DO NOTHING.
    d. Insert a Campaign record pointing at the existing channel/role, marked IsApproved.
    e. Scan the channel for existing threads by name; bind to found ones,
       create any of the five standard threads that are missing.
    f. Save AnnouncementsThreadID back to the campaign row.
    g. Bulk-insert CampaignPlayer rows for every member.
    h. Edit the deferred response with a summary.
*/

// importCampaignStandardThreads are the five threads Moontracer expects in every campaign channel.
var importCampaignStandardThreads = []string{"welcome", "announcements", "sessions", "dice-rolls", "general"}

// importCampaignThreadInitMessages maps thread names to their pinned welcome messages.
var importCampaignThreadInitMessages = map[string]string{
	"announcements": messages.ThreadInitMsgAnnouncements,
	"sessions":      messages.ThreadInitMsgSessions,
	"dice-rolls":    messages.ThreadInitMsgDiceRolls,
	"general":       messages.ThreadInitMsgGeneral,
}

type importCampaignCommand struct {
	db *bun.DB
}

func (c *importCampaignCommand) Data() *discordgo.ApplicationCommand {
	textChannelType := discordgo.ChannelTypeGuildText
	return &discordgo.ApplicationCommand{
		Name:        messages.ImportCampaignCommandName,
		Description: messages.ImportCampaignCommandDesc,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         messages.ImportCampaignOptChannel,
				Description:  "The existing campaign text channel.",
				Required:     true,
				ChannelTypes: []discordgo.ChannelType{textChannelType},
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        messages.ImportCampaignOptRole,
				Description: "The Discord role tied to this campaign.",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        messages.ImportCampaignOptDM,
				Description: "The Dungeon Master of this campaign.",
				Required:    true,
			},
		},
	}
}

func (c *importCampaignCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	invokerID := i.Member.User.ID
	guildID := i.GuildID

	ok, err := auth.Authorize(c.db, invokerID, auth.ScopeMod, "")
	if err != nil {
		log.Printf("importcampaign: auth check failed: %v", err)
		respond(s, i, messages.GenericErrorMessage)
		return
	}
	if !ok {
		respond(s, i, messages.AddPlayerNotDMOrModMessage)
		return
	}

	opts := i.ApplicationCommandData().Options
	channelID := opts[0].ChannelValue(s).ID
	roleID := opts[1].RoleValue(s, guildID).ID
	dmUser := opts[2].UserValue(s)
	dmID := dmUser.ID

	/*
		Discord gives 3 seconds for the bot to respond to a query, else it shows a failure message.
		This immediately acknowledges the requests and defers its execution with an Importing... message.
	*/
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: messages.ImportCampaignProcessing,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("importcampaign: deferred ack failed: %v", err)
		return
	}

	go func() {
		edit := func(content string) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
		}

		// 1. Read channel metadata.
		ch, err := s.Channel(channelID)
		if err != nil {
			log.Printf("importcampaign: fetch channel %s: %v", channelID, err)
			edit(messages.ImportCampaignErrChannel)
			return
		}
		channelName := ch.Name
		categoryID := ch.ParentID

		// 2. Collect all guild members who hold the campaign role.
		memberIDs := collectMembersWithRole(s, guildID, roleID)

		// 3. Ensure the DM is always included, even if they don't have the role yet.
		if !containsString(memberIDs, dmID) {
			memberIDs = append(memberIDs, dmID)
		}

		// 4. Create Player rows for everyone who doesn't have one.
		if err := models.BulkUpsertPlayers(c.db, memberIDs); err != nil {
			log.Printf("importcampaign: bulk upsert players: %v", err)
		}

		// 5. Build and insert the campaign record.
		campaign := &models.Campaign{
			ID:            uuid.NewString(),
			Name:          channelName,
			DungeonMaster: dmID,
			ChannelID:     channelID,
			CategoryID:    categoryID,
			RoleID:        roleID,
			IsApproved:    true,
			IsOpen:        true,
		}
		if _, err := c.db.NewInsert().Model(campaign).Exec(context.Background()); err != nil {
			log.Printf("importcampaign: insert campaign: %v", err)
			edit(messages.ImportCampaignErrDB)
			return
		}

		// 6. Wire up threads: bind existing, create missing.
		created, bound := bindOrCreateImportThreads(s, guildID, campaign, channelID, channelName)

		// 7. Persist AnnouncementsThreadID if it was resolved.
		if campaign.AnnouncementsThreadID != "" {
			if _, err := c.db.NewUpdate().Model(campaign).
				Column("announcements_thread_id").
				Where("id = ?", campaign.ID).
				Exec(context.Background()); err != nil {
				log.Printf("importcampaign: save announcements thread id: %v", err)
			}
		}

		// 8. Create CampaignPlayer rows.
		if err := models.BulkAddCampaignMembers(c.db, campaign.ID, memberIDs); err != nil {
			log.Printf("importcampaign: bulk add members: %v", err)
		}

		log.Printf("importcampaign: imported %q (%s): %d members, %d threads bound, %d threads created",
			channelName, campaign.ID, len(memberIDs), bound, created)

		edit(fmt.Sprintf(messages.ImportCampaignSuccess, channelName, len(memberIDs), bound, created))
	}()
}

func (c *importCampaignCommand) Hidden() bool { return true }

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

/*
bindOrCreateImportThreads checks for existing threads matching Moontracer's standard names,
binds to them if found, and creates any that are missing.
*/
func bindOrCreateImportThreads(s *discordgo.Session, guildID string, c *models.Campaign, channelID, channelName string) (created, bound int) {
	// Build a lookup of existing thread names in this channel.
	existing := map[string]string{} // threadName -> threadID

	if active, err := s.GuildThreadsActive(guildID); err == nil {
		for _, t := range active.Threads {
			if t.ParentID == channelID {
				existing[t.Name] = t.ID
			}
		}
	} else {
		log.Printf("importcampaign: fetch active threads: %v", err)
	}

	// Also check public archived threads for this channel.
	if archived, err := s.ThreadsArchived(channelID, nil, 100); err == nil {
		for _, t := range archived.Threads {
			if _, already := existing[t.Name]; !already {
				existing[t.Name] = t.ID
			}
		}
	}

	for _, name := range importCampaignStandardThreads {
		threadName := fmt.Sprintf("%s-%s", channelName, name)

		if id, found := existing[threadName]; found {
			if name == "announcements" {
				c.AnnouncementsThreadID = id
			}
			bound++
			continue
		}

		// Thread doesn't exist: create it.
		const archiveDuration = 10080 // 1 week in minutes
		thread, err := guard.ThreadStart(s, channelID, threadName, archiveDuration)
		if err != nil {
			log.Printf("importcampaign: create thread %s: %v", threadName, err)
			continue
		}
		created++

		if name == "announcements" {
			c.AnnouncementsThreadID = thread.ID
		}

		var initMsg string
		if name == "welcome" {
			initMsg = fmt.Sprintf(messages.ThreadInitMsgWelcomeFmt, c.Name)
		} else if msg, ok := importCampaignThreadInitMessages[name]; ok {
			initMsg = msg
		}

		if initMsg == "" {
			continue
		}
		msg, err := guard.ChannelMessageSend(s, thread.ID, initMsg)
		if err != nil {
			log.Printf("importcampaign: init message for %s: %v", threadName, err)
			continue
		}
		if err := guard.ChannelMessagePin(s, thread.ID, msg.ID); err != nil {
			log.Printf("importcampaign: pin in %s: %v", threadName, err)
		}
		if name == "welcome" {
			if err := guard.LockThread(s, thread.ID); err != nil {
				log.Printf("importcampaign: lock %s: %v", threadName, err)
			}
		}
	}
	return
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
