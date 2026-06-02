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

	Campaign  *Campaign           `bun:"rel:belongs-to,join:campaign_id=id"`
	Responses []SessionAssistance `bun:"rel:has-many,join:id=session_id"`
}

// SessionAssistance records a player's response for a specific session slot.
type SessionAssistance struct {
	bun.BaseModel `bun:"table:session_responses"`

	SessionID string         `bun:",pk,notnull"`
	PlayerID  string         `bun:",pk,notnull"`
	Status    ResponseStatus `bun:",notnull,default:''"`
	UpdatedAt time.Time      `bun:",nullzero"`

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

/*
GetSessionsThisWeek returns all upcoming sessions from approved, non-archived campaigns
scheduled within the next 7 days, sorted soonest first.

Used by /thisweek.
*/
func GetSessionsThisWeek(db *bun.DB) ([]Session, error) {
	ctx := context.Background()
	var sessions []Session
	err := db.NewSelect().Model(&sessions).
		Relation("Campaign").
		Join("JOIN campaigns AS cam ON cam.id = session.campaign_id").
		Where("cam.is_approved = ?", true).
		Where("cam.is_archived = ?", false).
		Where("session.status = ? AND session.scheduled_at > ? AND session.scheduled_at <= ?",
			SessionUpcoming, time.Now().UTC(), time.Now().UTC().Add(7*24*time.Hour)).
		OrderExpr("session.scheduled_at ASC").
		Scan(ctx)
	return sessions, err
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
		Relation("Campaign").
		Join("JOIN campaign_players AS cp ON cp.campaign_id = session.campaign_id").
		Join("JOIN campaigns AS cam ON cam.id = session.campaign_id").
		Where("cp.player_id = ? AND cp.status = ?", playerID, StatusActive).
		Where("cam.is_approved = ?", true).
		Where("session.status = ? AND session.scheduled_at > ?", SessionUpcoming, time.Now().UTC()).
		OrderExpr("session.scheduled_at ASC").
		Scan(ctx)
	return sessions, err
}

// GetSessionConfirmations returns all responses for a session slot.
func GetSessionConfirmations(db *bun.DB, sessionID string) ([]SessionAssistance, error) {
	ctx := context.Background()
	var rsvps []SessionAssistance
	err := db.NewSelect().Model(&rsvps).
		Where("session_id = ?", sessionID).
		Scan(ctx)
	return rsvps, err
}

// GetPlayerSessionConfirmation returns the response for a specific player and session, or nil if not found.
func GetPlayerSessionConfirmation(db *bun.DB, sessionID, playerID string) (*SessionAssistance, error) {
	ctx := context.Background()
	r := &SessionAssistance{}
	err := db.NewSelect().Model(r).
		Where("session_id = ? AND player_id = ?", sessionID, playerID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UpsertSessionPlayers creates or updates a player's response for a session.
func UpsertSessionPlayers(db *bun.DB, sessionID, playerID string, status ResponseStatus) error {
	ctx := context.Background()
	r := &SessionAssistance{
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

// CountAcceptedPlayers returns the number of players who accepted a session (not waitlisted).
func CountAcceptedPlayers(db *bun.DB, sessionID string) (int, error) {
	ctx := context.Background()
	count, err := db.NewSelect().Model((*SessionAssistance)(nil)).
		Where("session_id = ? AND status = ?", sessionID, ResponseAccepted).
		Count(ctx)
	return count, err
}

// GetFirstWaitlistedPlayer returns the earliest waitlisted player in the queue for a session, or nil if none.
func GetFirstWaitlistedPlayer(db *bun.DB, sessionID string) (*SessionAssistance, error) {
	r := &SessionAssistance{}
	err := db.NewSelect().Model(r).
		Where("session_id = ? AND status = ?", sessionID, ResponseWaitlisted).
		OrderExpr("updated_at ASC").
		Limit(1).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return r, nil
}

/*
GetPlayerConflictingSessions returns upcoming sessions the player has accepted
that overlap the given time window (~2 hours).
*/
func GetPlayerConflictingSessions(db *bun.DB, playerID string, at time.Time) ([]Session, error) {
	ctx := context.Background()
	windowStart := at.Add(-30 * time.Minute)
	windowEnd := at.Add(30 * time.Minute)

	var sessions []Session
	err := db.NewSelect().Model(&sessions).
		Join("JOIN session_responses sr ON sr.session_id = session.id").
		Where("sr.player_id = ? AND sr.status = ?", playerID, ResponseAccepted).
		Where("session.scheduled_at >= ? AND session.scheduled_at <= ?", windowStart, windowEnd).
		Where("session.status = ?", SessionUpcoming).
		OrderExpr("session.scheduled_at ASC").
		Scan(ctx)
	return sessions, err
}
