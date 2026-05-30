package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/importsession"
	"github.com/framebuffers/moontracer/internal/messages"
)

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
	guildID := i.GuildID
	opts := i.ApplicationCommandData().Options
	channelID := opts[0].ChannelValue(s).ID
	roleID := opts[1].RoleValue(s, guildID).ID
	dmID := opts[2].UserValue(s).ID

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
		edit := func(content string, components []discordgo.MessageComponent) {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    &content,
				Components: &components,
			})
		}

		ch, err := s.Channel(channelID)
		if err != nil {
			log.Printf("importcampaign: fetch channel %s: %v", channelID, err)
			edit(messages.ImportCampaignErrChannel, nil)
			return
		}

		threads := fetchExistingThreads(s, guildID, channelID)
		sessionID, _ := importsession.New(guildID, channelID, ch.Name, roleID, dmID, threads)

		content := fmt.Sprintf(messages.ImportStep1Header, ch.Name)
		edit(content, importsession.BuildStep1Components(sessionID, threads, nil))
	}()
}

func (c *importCampaignCommand) Hidden() bool { return true }

// fetchExistingThreads returns all active and archived threads in channelID.
func fetchExistingThreads(s *discordgo.Session, guildID, channelID string) []importsession.ThreadOption {
	seen := map[string]bool{}
	var opts []importsession.ThreadOption

	add := func(t *discordgo.Channel) {
		if t.ParentID == channelID && !seen[t.ID] {
			seen[t.ID] = true
			opts = append(opts, importsession.ThreadOption{ID: t.ID, Name: t.Name})
		}
	}

	if active, err := s.GuildThreadsActive(guildID); err == nil {
		for _, t := range active.Threads {
			add(t)
		}
	}
	if archived, err := s.ThreadsArchived(channelID, nil, 100); err == nil {
		for _, t := range archived.Threads {
			add(t)
		}
	}
	return opts
}
