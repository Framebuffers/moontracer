package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/interactions/helpers"
	"github.com/framebuffers/moontracer/internal/manager/models"
	"github.com/framebuffers/moontracer/internal/messages"
)

func fireReminder(s *Scheduler, guildID, campaignID string) {
	gdb, err := s.guildDBM.GetOrCreate(guildID)
	if err != nil {
		log.Printf("scheduler: fire: get db for guild %s: %v", guildID, err)
		return
	}

	campaign, err := db.GetByID[models.Campaign](gdb, campaignID)
	if err != nil {
		log.Printf("scheduler: fire: load campaign %s: %v", campaignID, err)
		return
	}

	// Re-validate reminder. State may have changed since the timer was set.
	if campaign.IsArchived || campaign.Schedule.AlertSent || campaign.Schedule.NextSession.IsZero() {
		return
	}
	if !campaign.Schedule.NextSession.After(time.Now().UTC()) {
		return
	}

	/*
		Persist the idempotency flag BEFORE fan-out, so a restart between
		here and the dispatcher push never sends duplicate reminders.
	*/
	campaign.Schedule.AlertSent = true
	if err := db.Update(gdb, campaign); err != nil {
		log.Printf("scheduler: fire: mark AlertSent for campaign %s: %v", campaignID, err)
		return
	}

	players, err := models.GetCampaignPlayers(gdb, campaignID)
	if err != nil {
		log.Printf("scheduler: fire: load players for campaign %s: %v", campaignID, err)
		return
	}

	sent := 0
	for _, p := range players {
		if p.Status != models.StatusActive &&
			p.Status != models.StatusPending &&
			p.Status != models.StatusHiatus {
			continue
		}
		settings, err := models.GetOrCreatePlayerSettings(gdb, p.PlayerID)
		if err != nil {
			log.Printf("scheduler: fire: load settings for player %s: %v", p.PlayerID, err)
			continue
		}
		if !settings.NotifySessionRemind {
			continue
		}
		loc := settings.Location()
		displayTime := helpers.FormatInLocation(campaign.Schedule.NextSession, messages.SessionTimeFormat, loc)
		content := fmt.Sprintf(messages.ReminderContent,
			campaign.Name,
			displayTime,
			helpers.TZLabel(loc),
		)
		content += formatReminderLinks(campaign, p.SheetURL)
		s.dispatcher.Push(dispatch.DirectMessage{
			ID:      fmt.Sprintf("reminder:%s:%s", campaignID, p.PlayerID),
			Target:  p.PlayerID,
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.RSVPAcceptLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.RSVPAcceptPrefix, guildID, campaignID),
					},
					discordgo.Button{
						Label:    messages.RSVPDeclineLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.RSVPDeclinePrefix, guildID, campaignID),
					},
				}},
			},
		})
		sent++
	}

	log.Printf("scheduler: fired reminder for %s (%s, guild %s) - %d DM(s) queued",
		campaign.Name, campaignID, guildID, sent)
}

// fireSessionReminder sends RSVP reminder DMs for a session-table session.
func fireSessionReminder(s *Scheduler, guildID, sessionID string) {
	gdb, err := s.guildDBM.GetOrCreate(guildID)
	if err != nil {
		log.Printf("scheduler: session fire: get db for guild %s: %v", guildID, err)
		return
	}

	session := &models.Session{}
	if err := gdb.NewSelect().Model(session).Where("id = ?", sessionID).Scan(context.Background()); err != nil {
		log.Printf("scheduler: session fire: load session %s: %v", sessionID, err)
		return
	}
	if session.AlertSent || session.Status != models.SessionUpcoming || !session.ScheduledAt.After(time.Now().UTC()) {
		return
	}

	session.AlertSent = true
	if _, err := gdb.NewUpdate().Model(session).Column("alert_sent").WherePK().Exec(context.Background()); err != nil {
		log.Printf("scheduler: session fire: mark alert_sent for %s: %v", sessionID, err)
		return
	}

	campaign, err := db.GetByID[models.Campaign](gdb, session.CampaignID)
	if err != nil {
		log.Printf("scheduler: session fire: load campaign %s: %v", session.CampaignID, err)
		return
	}

	players, err := models.GetCampaignPlayers(gdb, session.CampaignID)
	if err != nil {
		log.Printf("scheduler: session fire: load players for campaign %s: %v", session.CampaignID, err)
		return
	}

	sent := 0
	for _, p := range players {
		if p.Status != models.StatusActive && p.Status != models.StatusPending && p.Status != models.StatusHiatus {
			continue
		}
		settings, err := models.GetOrCreatePlayerSettings(gdb, p.PlayerID)
		if err != nil || !settings.NotifySessionRemind {
			continue
		}
		displayTime := helpers.FormatInLocation(session.ScheduledAt, messages.SessionTimeFormat, settings.Location())
		content := fmt.Sprintf(messages.SessionReminderContentFmt,
			campaign.Name,
			session.ScheduledAt.Unix(),
			formatReminderLinks(campaign, p.SheetURL),
		)
		_ = displayTime // shown via Discord <t:> timestamp in content
		s.dispatcher.Push(dispatch.DirectMessage{
			ID:      fmt.Sprintf("session-reminder:%s:%s", sessionID, p.PlayerID),
			Target:  p.PlayerID,
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    messages.SessionRSVPAcceptLabel,
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPAcceptPrefix, guildID, sessionID),
					},
					discordgo.Button{
						Label:    messages.SessionRSVPDeclineLabel,
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("%s:%s:%s", messages.SessionRSVPDeclinePrefix, guildID, sessionID),
					},
				}},
			},
		})
		sent++
	}
	log.Printf("scheduler: fired session reminder for %s (guild %s) - %d DM(s) queued", sessionID, guildID, sent)
}

func formatReminderLinks(c *models.Campaign, sheetURL string) string {
	hasVTT := c.VTTLink != ""
	hasSheets := sheetURL != ""
	hasResources := len(c.Links) > 0
	if !hasVTT && !hasSheets && !hasResources {
		return ""
	}
	var b strings.Builder
	b.WriteString(messages.ManageReminderLinks)
	if hasVTT {
		b.WriteString(fmt.Sprintf(messages.ManageReminderVTT, c.VTTLink))
	}
	if hasSheets {
		b.WriteString(fmt.Sprintf(messages.ManageReminderSheets, sheetURL))
	}
	for _, r := range c.Links {
		b.WriteString(fmt.Sprintf(messages.ManageReminderResource, r))
	}
	return b.String()
}
