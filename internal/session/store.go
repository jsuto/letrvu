package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session holds the credentials and IMAP connection details for one
// logged-in user. Credentials are kept in memory only — never written to disk.
type Session struct {
	ID        string
	IMAPHost  string
	IMAPPort  int
	SMTPHost  string
	SMTPPort  int
	Username  string
	Password  string
	CreatedAt time.Time
}

// Store is a thread-safe in-memory session registry.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session)}
}

func (s *Store) Create(imapHost string, imapPort int, smtpHost string, smtpPort int, username, password string) (*Session, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID:        id,
		IMAPHost:  imapHost,
		IMAPPort:  imapPort,
		SMTPHost:  smtpHost,
		SMTPPort:  smtpPort,
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess, nil
}

func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
