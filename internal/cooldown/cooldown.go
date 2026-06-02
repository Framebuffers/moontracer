package cooldown

import (
	"fmt"
	"sync"
	"time"
)

// Global is the process-wide cooldown store shared by all handlers.
var Global = New()

/*
Store is an in-memory rate-limiter for short-lived cooldowns.

It is safe for concurrent use. Keys are arbitrary strings composed by callers.
*/
type Store struct {
	mu      sync.Mutex
	timed   map[string]time.Time
	oneshot map[string]struct{}
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		timed:   make(map[string]time.Time),
		oneshot: make(map[string]struct{}),
	}
}

/*
Allow returns true if key is not on cooldown and starts a new cooldown of d.

Returns false (and does not reset the timer) if the existing cooldown has not expired.
*/
func (s *Store) Allow(key string, d time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if exp, ok := s.timed[key]; ok && time.Now().Before(exp) {
		return false
	}
	s.timed[key] = time.Now().Add(d)
	return true
}

/*
Remaining returns how long until key's cooldown expires.

Returns 0 if the key is not on cooldown or has already expired.
*/
func (s *Store) Remaining(key string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.timed[key]
	if !ok {
		return 0
	}
	r := time.Until(exp)
	if r < 0 {
		return 0
	}
	return r
}

// AllowOnce returns true the first time key is seen, false on every subsequent call.
func (s *Store) AllowOnce(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, used := s.oneshot[key]; used {
		return false
	}
	s.oneshot[key] = struct{}{}
	return true
}

// FormatRemaining formats a duration as a human-readable cooldown string, e.g. "14 minutes".
func FormatRemaining(d time.Duration) string {
	minutes := int(d.Minutes()) + 1
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}
