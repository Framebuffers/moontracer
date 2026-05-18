package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// SessionStatus tracks the lifecycle of a scheduled session slot.
type SessionStatus string

const (
	SessionUpcoming  SessionStatus = "upcoming"
	SessionComplete  SessionStatus = "complete"
	SessionCancelled SessionStatus = "cancelled"
)

// Session represents a single scheduled session slot for a campaign.
type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	ID           string        `bun:",pk,notnull"`
	CampaignID   string        `bun:",notnull"`
	ScheduledAt  time.Time     `bun:",notnull"`
	Title        string        `bun:",notnull,default:''"`
	ChannelMsgID string        `bun:",notnull,default:''"` // message ID of the announcement embed
	Capacity     int           `bun:",notnull,default:0"`  // 0 = use campaign capacity
	AlertSent    bool          `bun:",notnull,default:false"`
	Status       SessionStatus `bun:",notnull,default:'upcoming'"`
	CreatedAt    time.Time     `bun:",nullzero"`

	Campaign *Campaign     `bun:"rel:belongs-to,join:campaign_id=id"`
	RSVPs    []SessionRSVP `bun:"rel:has-many,join:id=session_id"`
}

// SessionRSVP records a player's RSVP for a specific session slot.
type SessionRSVP struct {
	bun.BaseModel `bun:"table:session_rsvps"`

	SessionID string     `bun:",pk,notnull"`
	PlayerID  string     `bun:",pk,notnull"`
	Status    RSVPStatus `bun:",notnull,default:''"`
	UpdatedAt time.Time  `bun:",nullzero"`

	Session *Session `bun:"rel:belongs-to,join:session_id=id"`
}

// NewSession creates a Session with a generated ID and sets CreatedAt.
func NewSession(campaignID string, scheduledAt time.Time, title string, capacity int) *Session {
	return &Session{
		ID:          uuid.NewString(),
		CampaignID:  campaignID,
		ScheduledAt: scheduledAt,
		Title:       title,
		Capacity:    capacity,
		Status:      SessionUpcoming,
		CreatedAt:   time.Now().UTC(),
	}
}

// GetUpcomingSessions returns all upcoming sessions for a campaign, sorted soonest first.
func GetUpcomingSessions(db *bun.DB, campaignID string) ([]Session, error) {
	ctx := context.Background()
	var sessions []Session
	err := db.NewSelect().Model(&sessions).
		Where("campaign_id = ? AND status = ? AND scheduled_at > ?", campaignID, SessionUpcoming, time.Now().UTC()).
		OrderExpr("scheduled_at ASC").
		Scan(ctx)
	return sessions, err
}

/*
GetAllUpcomingSessionsForPlayer returns all upcoming sessions across all campaigns
where the player is an active member.
*/
func GetAllUpcomingSessionsForPlayer(db *bun.DB, playerID string) ([]Session, error) {
	ctx := context.Background()
	var sessions []Session
	err := db.NewSelect().Model(&sessions).
		Join("JOIN campaign_players cp ON cp.campaign_id = session.campaign_id").
		Where("cp.player_id = ? AND cp.status = ?", playerID, StatusActive).
		Where("session.status = ? AND session.scheduled_at > ?", SessionUpcoming, time.Now().UTC()).
		OrderExpr("session.scheduled_at ASC").
		Scan(ctx)
	return sessions, err
}

// GetSessionRSVPs returns all RSVPs for a session slot.
func GetSessionRSVPs(db *bun.DB, sessionID string) ([]SessionRSVP, error) {
	ctx := context.Background()
	var rsvps []SessionRSVP
	err := db.NewSelect().Model(&rsvps).
		Where("session_id = ?", sessionID).
		Scan(ctx)
	return rsvps, err
}

// GetPlayerSessionRSVP returns the RSVP for a specific player and session, or nil if not found.
func GetPlayerSessionRSVP(db *bun.DB, sessionID, playerID string) (*SessionRSVP, error) {
	ctx := context.Background()
	r := &SessionRSVP{}
	err := db.NewSelect().Model(r).
		Where("session_id = ? AND player_id = ?", sessionID, playerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UpsertSessionRSVP creates or updates a player's RSVP for a session.
func UpsertSessionRSVP(db *bun.DB, sessionID, playerID string, status RSVPStatus) error {
	ctx := context.Background()
	r := &SessionRSVP{
		SessionID: sessionID,
		PlayerID:  playerID,
		Status:    status,
		UpdatedAt: time.Now().UTC(),
	}
	_, err := db.NewInsert().Model(r).
		On("CONFLICT (session_id, player_id) DO UPDATE SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

// CountAcceptedRSVPs returns the number of players who accepted a session (not waitlisted).
func CountAcceptedRSVPs(db *bun.DB, sessionID string) (int, error) {
	ctx := context.Background()
	count, err := db.NewSelect().Model((*SessionRSVP)(nil)).
		Where("session_id = ? AND status = ?", sessionID, RSVPAccepted).
		Count(ctx)
	return count, err
}

/*
GetPlayerConflictingSessions returns upcoming sessions the player has accepted
that overlap the given time window (~2 hours).
*/
func GetPlayerConflictingSessions(db *bun.DB, playerID string, at time.Time) ([]Session, error) {
	ctx := context.Background()
	windowStart := at.Add(-2 * time.Hour)
	windowEnd := at.Add(2 * time.Hour)

	var sessions []Session
	err := db.NewSelect().Model(&sessions).
		Join("JOIN session_rsvps sr ON sr.session_id = session.id").
		Where("sr.player_id = ? AND sr.status = ?", playerID, RSVPAccepted).
		Where("session.scheduled_at >= ? AND session.scheduled_at <= ?", windowStart, windowEnd).
		Where("session.status = ?", SessionUpcoming).
		OrderExpr("session.scheduled_at ASC").
		Scan(ctx)
	return sessions, err
}
