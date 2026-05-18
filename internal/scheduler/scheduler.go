package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/framebuffers/moontracer/internal/db"
	"github.com/framebuffers/moontracer/internal/dispatch"
	"github.com/framebuffers/moontracer/internal/manager/models"
)

const DefaultLeadTime = time.Hour

// Scheduler manages one-shot session reminder timers, keyed by guild/campaign.
type Scheduler struct {
	guildDBM   *db.GuildDBManager
	dispatcher *dispatch.Dispatcher

	mu     sync.Mutex
	timers map[string]*time.Timer
}

func New(guildDBM *db.GuildDBManager, dispatcher *dispatch.Dispatcher) *Scheduler {
	return &Scheduler{
		guildDBM:   guildDBM,
		dispatcher: dispatcher,
		timers:     make(map[string]*time.Timer),
	}
}

func timerKey(guildID, id string) string { return guildID + "/" + id }

/*
BootScan re-schedules reminders for upcoming sessions (sessions table) and any
legacy campaigns that still have NextSession set outside the sessions table.

Call once after all guild DBs are initialised.
*/
func (s *Scheduler) BootScan(guildIDs []string) {
	now := time.Now().UTC()
	total := 0
	for _, guildID := range guildIDs {
		gdb, err := s.guildDBM.GetOrCreate(guildID)
		if err != nil {
			log.Printf("scheduler: boot scan: get db for guild %s: %v", guildID, err)
			continue
		}

		// Session-table reminders (new path).
		var sessions []models.Session
		if err := gdb.NewSelect().Model(&sessions).
			Where("status = ? AND scheduled_at > ? AND alert_sent = 0", models.SessionUpcoming, now).
			Scan(context.Background()); err != nil {
			log.Printf("scheduler: boot scan sessions for guild %s: %v", guildID, err)
		}
		for i := range sessions {
			s.ScheduleSession(guildID, &sessions[i])
			total++
		}

		/*
			LEGACY CODE:
			Legacy campaign-level reminders (campaigns that have NextSession set but
			no corresponding session row yet: they were not seeded by migration).
		*/
		campaigns, err := db.GetAll[models.Campaign](gdb)
		if err != nil {
			log.Printf("scheduler: boot scan campaigns for guild %s: %v", guildID, err)
			continue
		}
		for i := range campaigns {
			c := &campaigns[i]
			if c.IsArchived || c.Schedule.AlertSent || c.Schedule.NextSession.IsZero() {
				continue
			}
			if !c.Schedule.NextSession.After(now) {
				continue
			}
			// Skip if a session row covers this (to avoid duplicate reminders).
			var count int
			count, _ = gdb.NewSelect().Model((*models.Session)(nil)).
				Where("campaign_id = ? AND scheduled_at = ? AND status = ?",
					c.ID, c.Schedule.NextSession, models.SessionUpcoming).
				Count(context.Background())
			if count > 0 {
				continue
			}
			s.Schedule(guildID, c)
			total++
		}
	}
	log.Printf("scheduler: boot scan complete - %d reminder(s) scheduled across %d guild(s)", total, len(guildIDs))
}

/*
Schedule cancels any existing timer for the campaign and sets a new one.

Safe to call after every NextSession update. It always replaces the prior timer.
*/
func (s *Scheduler) Schedule(guildID string, campaign *models.Campaign) {
	if s == nil {
		return
	}
	if campaign.IsArchived || campaign.Schedule.AlertSent || campaign.Schedule.NextSession.IsZero() {
		return
	}
	now := time.Now().UTC()
	if !campaign.Schedule.NextSession.After(now) {
		return
	}

	delay := time.Until(campaign.Schedule.NextSession.Add(-DefaultLeadTime))
	if delay < 0 {
		// NOTE: it's past lead-time window but session hasn't started. fire alert immediately.
		delay = 0
	}

	key := timerKey(guildID, campaign.ID)
	campaignID := campaign.ID

	s.mu.Lock()
	if existing, ok := s.timers[key]; ok {
		existing.Stop()
	}
	s.timers[key] = time.AfterFunc(delay, func() {
		s.mu.Lock()
		delete(s.timers, key)
		s.mu.Unlock()
		fireReminder(s, guildID, campaignID)
	})
	s.mu.Unlock()

	log.Printf("scheduler: reminder for campaign %s (guild %s) in %v", campaignID, guildID, delay.Truncate(time.Second))
}

/*
ScheduleSession sets a session-level reminder timer.

Uses timerKey(guildID, session.ID) so it never collides with campaign-level timers.
*/
func (s *Scheduler) ScheduleSession(guildID string, session *models.Session) {
	if s == nil {
		return
	}
	if session.AlertSent || session.Status != models.SessionUpcoming {
		return
	}
	now := time.Now().UTC()
	if !session.ScheduledAt.After(now) {
		return
	}

	delay := time.Until(session.ScheduledAt.Add(-DefaultLeadTime))
	if delay < 0 {
		delay = 0
	}

	key := timerKey(guildID, session.ID)
	sessionID := session.ID

	s.mu.Lock()
	if existing, ok := s.timers[key]; ok {
		existing.Stop()
	}
	s.timers[key] = time.AfterFunc(delay, func() {
		s.mu.Lock()
		delete(s.timers, key)
		s.mu.Unlock()
		fireSessionReminder(s, guildID, sessionID)
	})
	s.mu.Unlock()

	log.Printf("scheduler: session reminder for %s (guild %s) in %v", sessionID, guildID, delay.Truncate(time.Second))
}

/*
Cancel stops the pending timer for a campaign without firing it.

Call whenever a campaign is archived or its session is cleared.
*/
func (s *Scheduler) Cancel(guildID, campaignID string) {
	if s == nil {
		return
	}
	key := timerKey(guildID, campaignID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[key]; ok {
		t.Stop()
		delete(s.timers, key)
		log.Printf("scheduler: cancelled reminder for campaign %s (guild %s)", campaignID, guildID)
	}
}

// Stop cancels all pending timers. Call during bot shutdown.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, t := range s.timers {
		t.Stop()
		delete(s.timers, key)
	}
	log.Printf("scheduler: stopped")
}
