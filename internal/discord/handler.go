package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/commands"
	"github.com/framebuffers/moontracer/internal/guard"
)

// NewHandler returns a discordgo event handler that dispatches slash commands.
func NewHandler(cmds []commands.Command) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("handler: panic recovered: %v", r)
				respondEphemeral(s, i, "An unexpected error occurred. Please try again.")
			}
		}()

		guildID := i.GuildID
		if guildID == "" {
			respondEphemeral(s, i, "This command must be used in a server.")
			return
		}

		if guard.DebugGuildID != "" && guildID != guard.DebugGuildID {
			log.Printf("handler: rejecting interaction from guild %s (scoped to %s)", guildID, guard.DebugGuildID)
			respondEphemeral(s, i, "This bot is scoped to a single server and will not respond here.")
			return
		}

		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

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
				cmd.Execute(s, i)
				return
			}
		}
		log.Printf("handler: unknown command: /%s", name)
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
