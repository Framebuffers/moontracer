package auditlog

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"

	"github.com/framebuffers/moontracer/internal/guard"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

/*
Post writes an audit entry and, if AuditLogChannelID is configured, posts an embed to it.

Discord posting runs in the background and never blocks the caller.
*/
func Post(s *discordgo.Session, database *bun.DB, guildID, playerID, authorID string, action models.AuditAction, reason string) {
	if err := models.InsertAuditEntry(database, playerID, authorID, action, reason); err != nil {
		log.Printf("auditlog: insert entry [%s] player=%s author=%s: %v", action, playerID, authorID, err)
	}
	go postEmbed(s, database, guildID, playerID, authorID, action, reason)
}

func postEmbed(s *discordgo.Session, database *bun.DB, guildID, playerID, authorID string, action models.AuditAction, reason string) {
	settings, err := models.GetOrCreateGuildSettings(database)
	if err != nil || !messages.IsSnowflake(settings.AuditLogChannelID) {
		return
	}

	desc := fmt.Sprintf("**Action:** `%s`\n**By:** <@%s>\n**Subject:** <@%s>", action, authorID, playerID)
	if reason != "" {
		desc += "\n**Reason:** " + reason
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📋 Audit Event",
		Description: desc,
		Color:       0xFF4444,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Guild: " + guildID},
	}

	if _, err := guard.ChannelMessageSendComplex(s, settings.AuditLogChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	}); err != nil {
		log.Printf("auditlog: post embed to %s: %v", settings.AuditLogChannelID, err)
	}
}
