package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]int64
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]int64)}
}

func (s *SessionStore) Create(userID int64) (string, error) {
	id, err := randomToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.sessions[id] = userID
	s.mu.Unlock()
	return id, nil
}

func (s *SessionStore) User(sessionID string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.sessions[sessionID]
	return userID, ok
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
