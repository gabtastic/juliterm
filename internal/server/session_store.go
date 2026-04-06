package server

import (
	"sync"
	"time"
)

// AuthSession represents an authenticated session in the server-side store.
type AuthSession struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore is a thread-safe in-memory store for authenticated sessions.
// It is intentionally ephemeral — a server restart clears the store, which is
// consistent with the sessionSecret also being regenerated on restart.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*AuthSession
	ttl      time.Duration
}

// NewSessionStore creates a store with the given session TTL.
func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*AuthSession),
		ttl:      ttl,
	}
}

// Add registers a new session. If the ID already exists it is overwritten.
func (s *SessionStore) Add(id string) {
	now := time.Now()
	s.mu.Lock()
	s.sessions[id] = &AuthSession{
		ID:        id,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}
	s.mu.Unlock()
}

// Get returns the session if it exists and has not expired. Returns nil otherwise.
func (s *SessionStore) Get(id string) *AuthSession {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}
	if time.Now().After(sess.ExpiresAt) {
		// Lazy eviction of expired session.
		s.Delete(id)
		return nil
	}
	return sess
}

// Delete removes a session by ID. It is a no-op if the session does not exist.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// DeleteAll removes every session from the store.
func (s *SessionStore) DeleteAll() {
	s.mu.Lock()
	s.sessions = make(map[string]*AuthSession)
	s.mu.Unlock()
}

// Sweep removes all expired sessions and returns their IDs so callers can
// take further action (e.g. closing associated WebSocket connections).
func (s *SessionStore) Sweep() []string {
	now := time.Now()
	var expired []string

	s.mu.Lock()
	for id, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			expired = append(expired, id)
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()

	return expired
}
