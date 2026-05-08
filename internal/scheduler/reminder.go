package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"moontracer/internal/db"
	"moontracer/internal/dispatch"
	"moontracer/internal/interactions/helpers"
	"moontracer/internal/manager/models"
	"moontracer/internal/messages"
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
		content += formatReminderLinks(campaign)
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

	log.Printf("scheduler: fired reminder for %s (%s, guild %s) — %d DM(s) queued",
		campaign.Name, campaignID, guildID, sent)
}

func formatReminderLinks(c *models.Campaign) string {
	hasVTT := c.VTTLink != ""
	hasSheets := c.PlayerSheetURL != ""
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
		b.WriteString(fmt.Sprintf(messages.ManageReminderSheets, c.PlayerSheetURL))
	}
	for _, r := range c.Links {
		b.WriteString(fmt.Sprintf(messages.ManageReminderResource, r))
	}
	return b.String()
}
