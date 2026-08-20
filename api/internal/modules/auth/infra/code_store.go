package infra

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
)

const (
	codeBytes       = 32
	codeTTL         = 60 * time.Second
	cleanupInterval = 30 * time.Second
)

// ErrCodeInvalid is returned when the exchange code is not found or has expired.
var ErrCodeInvalid = errors.New("auth.invalid_exchange_code")

type codeEntry struct {
	tokens    *domain.TokenPair
	expiresAt time.Time
}

// AuthCodeStore is a thread-safe in-memory store for one-time auth exchange codes.
// Codes are generated after OAuth callback and exchanged for tokens by the frontend.
type AuthCodeStore struct {
	mu    sync.Mutex
	codes map[string]*codeEntry
	done  chan struct{}
}

// NewAuthCodeStore creates a new store and starts a background goroutine
// that removes expired codes every 30 seconds.
func NewAuthCodeStore() *AuthCodeStore {
	s := &AuthCodeStore{
		codes: make(map[string]*codeEntry),
		done:  make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Store saves the token pair and returns a random one-time code (64 hex chars).
func (s *AuthCodeStore) Store(tokens *domain.TokenPair) (string, error) {
	b := make([]byte, codeBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := hex.EncodeToString(b)

	s.mu.Lock()
	s.codes[code] = &codeEntry{
		tokens:    tokens,
		expiresAt: time.Now().Add(codeTTL),
	}
	s.mu.Unlock()

	return code, nil
}

// Exchange validates the code, returns the associated tokens, and deletes
// the code so it cannot be reused.
func (s *AuthCodeStore) Exchange(code string) (*domain.TokenPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.codes[code]
	if !ok {
		return nil, ErrCodeInvalid
	}

	delete(s.codes, code)

	if time.Now().After(entry.expiresAt) {
		return nil, ErrCodeInvalid
	}

	return entry.tokens, nil
}

// Stop signals the background cleanup goroutine to exit.
func (s *AuthCodeStore) Stop() {
	close(s.done)
}

func (s *AuthCodeStore) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.removeExpired()
		case <-s.done:
			return
		}
	}
}

func (s *AuthCodeStore) removeExpired() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for code, entry := range s.codes {
		if now.After(entry.expiresAt) {
			delete(s.codes, code)
		}
	}
}
