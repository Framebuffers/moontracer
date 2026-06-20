package importsession

import (
	"sync"

	"github.com/google/uuid"

	"github.com/framebuffers/moontracer/internal/messages"
)

// ThreadOption is a Discord thread the bot found in the channel at import time.
type ThreadOption struct {
	ID   string
	Name string
}

/*
Session holds an unfinished /importcampaign interaction.

It lives in memory for the lifetime of the two steps of the select menu flow.
It's deleted when the user confirms or cancels.
*/
type Session struct {
	GuildID     string
	ChannelID   string
	ChannelName string
	RoleID      string
	DMID        string

	// ExistingThreads is fetched once from Discord and presented as select-menu options.
	ExistingThreads []ThreadOption

	mu       sync.Mutex
	Mappings map[string]string
}

// SetThreadMapping stores a thread mapping. This is safe for concurrent use by select handlers.
func (s *Session) SetThreadMapping(threadName, value string) {
	s.mu.Lock()
	s.Mappings[threadName] = value
	s.mu.Unlock()
}

// GetCurrentThreadName returns the current value for a thread name (messages.ImportCreateNew if unset).
func (s *Session) GetCurrentThreadName(threadName string) string {
	s.mu.Lock()
	v, ok := s.Mappings[threadName]
	s.mu.Unlock()
	if !ok {
		return messages.ImportCreateNew
	}
	return v
}

var (
	storeMu sync.Mutex
	store   = map[string]*Session{}
)

// New creates a session, stores it, and returns its ID.
func New(guildID, channelID, channelName, roleID, dmID string, threads []ThreadOption) (string, *Session) {
	id := uuid.NewString()
	s := &Session{
		GuildID:         guildID,
		ChannelID:       channelID,
		ChannelName:     channelName,
		RoleID:          roleID,
		DMID:            dmID,
		ExistingThreads: threads,
		Mappings:        make(map[string]string),
	}
	storeMu.Lock()
	store[id] = s
	storeMu.Unlock()
	return id, s
}

// Get retrieves a session by ID.
func Get(id string) (*Session, bool) {
	storeMu.Lock()
	s, ok := store[id]
	storeMu.Unlock()
	return s, ok
}

// Delete removes a session by ID.
func Delete(id string) {
	storeMu.Lock()
	delete(store, id)
	storeMu.Unlock()
}
