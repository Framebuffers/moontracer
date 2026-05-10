package discord

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"moontracer/internal/commands"
	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/guard"
	"moontracer/internal/interactions"
	"moontracer/internal/scheduler"
)

/*
	Flow:
		1. NewHandler receives the GuildDBManager, dispatcher, and admin role name.
		2. Returns a closure that acts as the main Discord event handler.
		3. For each interaction:
			a. Resolve the guild ID: from the interaction itself, or from the
			   CustomID for DM interactions (approval buttons encode guild ID).
			b. Look up or create the guild's database via GuildDBManager.
			c. Build handler sets for that guild's DB.
			d. Dispatch to the appropriate handler.
*/

/*
NewHandler returns a discordgo event handler that resolves the guild's
database per interaction, then dispatches slash commands, component interactions (buttons), and modal submissions.
*/
func NewHandler(
	guildDBM *db.GuildDBManager,
	dispatcher *dispatch.Dispatcher,
	adminRole string,
	sched *scheduler.Scheduler,
	dataDir, mediaBaseURL string,
) func(s *discordgo.Session, i *discordgo.InteractionCreate) {

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		guildID := i.GuildID

		/*
			Note:
				DM interactions have no guild ID.

				For approval buttons/modals sent via DM, the guild ID is encoded in
				the CustomID as the second segment (prefix:guildID:campaignID).
		*/
		if guildID == "" {
			guildID = extractGuildFromCustomID(i)
		}
		if guildID == "" {
			respondEphemeral(s, i, "This command must be used in a server.")
			return
		}

		if guard.DebugGuildID != "" && guildID != guard.DebugGuildID {
			log.Printf("handler: rejecting interaction from guild %s (scoped to %s)", guildID, guard.DebugGuildID)
			respondEphemeral(s, i, "This bot is scoped to a single server and will not respond here.")
			return
		}

		guildDB, err := guildDBM.GetOrCreate(guildID)
		if err != nil {
			log.Printf("handler: failed to get DB for guild %s: %v", guildID, err)
			respondEphemeral(s, i, "Internal error — please try again later.")
			return
		}

		cmds := commands.All(guildDB, dispatcher, dataDir, mediaBaseURL)
		components := interactions.AllComponents(guildDB, dispatcher, sched, dataDir, mediaBaseURL)
		modals := interactions.AllModals(guildDB, dispatcher, sched, dataDir, mediaBaseURL)

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			name := i.ApplicationCommandData().Name
			for _, cmd := range cmds {
				if cmd.Data().Name == name {
					userID := "unknown"
					if i.Member != nil {
						userID = i.Member.User.ID
					} else if i.User != nil {
						userID = i.User.ID
					}
					log.Printf("handler: /%s invoked by %s in guild %s", name, userID, guildID)
					incrementCommandCounter(guildDB, name)
					cmd.Execute(s, i)
					return
				}
			}
			log.Printf("handler: unknown command: /%s", name)

		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			prefix := customID
			if idx := strings.Index(customID, ":"); idx != -1 {
				prefix = customID[:idx]
			}
			for _, c := range components {
				if c.CustomIDPrefix() == prefix {
					log.Printf("handler: component %s triggered in guild %s", customID, guildID)
					c.HandleComponents(s, i)
					return
				}
			}
			log.Printf("handler: unknown component: %s", customID)

		case discordgo.InteractionApplicationCommandAutocomplete:
			name := i.ApplicationCommandData().Name
			for _, cmd := range cmds {
				if cmd.Data().Name == name {
					if ac, ok := cmd.(commands.AutocompleteCommand); ok {
						ac.Autocomplete(s, i)
						return
					}
					break
				}
			}

		case discordgo.InteractionModalSubmit:
			customID := i.ModalSubmitData().CustomID
			prefix := customID
			if idx := strings.Index(customID, ":"); idx != -1 {
				prefix = customID[:idx]
			}
			for _, m := range modals {
				if m.CustomIDPrefix() == prefix {
					log.Printf("handler: modal %s submitted in guild %s", customID, guildID)
					m.HandleModal(s, i)
					return
				}
			}
			log.Printf("handler: unknown modal: %s", customID)
		}
	}
}

/*
extractGuildFromCustomID attempts to extract a guild ID from a DM
interaction's CustomID.

Approval buttons use the format prefix:<guildID>:<campaignID>.
*/
func extractGuildFromCustomID(i *discordgo.InteractionCreate) string {
	var customID string
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID = i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		customID = i.ModalSubmitData().CustomID
	default:
		return ""
	}

	parts := strings.SplitN(customID, ":", 3)
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

/*
incrementCommandCounter bumps times_used for the given command name.

Runs synchronously so command handlers observe the new value if they read it.
Logs and swallows errors, since a counter bump must not block command execution.
*/
func incrementCommandCounter(guildDB *bun.DB, name string) {
	_, err := guildDB.NewUpdate().
		Table("commands").
		Set("times_used = times_used + 1").
		Where("name = ?", name).
		Exec(context.Background())
	if err != nil {
		log.Printf("handler: failed to increment counter for /%s: %v", name, err)
	}
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
