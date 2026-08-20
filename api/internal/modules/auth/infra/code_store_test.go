package infra

import (
	"errors"
	"testing"
	"time"

	"github.com/genkovich/task-tracker/api/internal/modules/auth/domain"
)

func testTokenPair() *domain.TokenPair {
	return &domain.TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    900,
	}
}

func TestAuthCodeStore_StoreReturnsUniqueCode(t *testing.T) {
	store := NewAuthCodeStore()
	defer store.Stop()

	codes := make(map[string]struct{})
	for range 100 {
		code, err := store.Store(testTokenPair())
		if err != nil {
			t.Fatalf("Store() error: %v", err)
		}
		if len(code) != 64 {
			t.Fatalf("expected 64 hex chars, got %d", len(code))
		}
		if _, exists := codes[code]; exists {
			t.Fatalf("duplicate code generated: %s", code)
		}
		codes[code] = struct{}{}
	}
}

func TestAuthCodeStore_ExchangeReturnsTokens(t *testing.T) {
	store := NewAuthCodeStore()
	defer store.Stop()

	tokens := testTokenPair()
	code, err := store.Store(tokens)
	if err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	got, err := store.Exchange(code)
	if err != nil {
		t.Fatalf("Exchange() error: %v", err)
	}

	if got.AccessToken != tokens.AccessToken {
		t.Fatalf("AccessToken: got %q, want %q", got.AccessToken, tokens.AccessToken)
	}
	if got.RefreshToken != tokens.RefreshToken {
		t.Fatalf("RefreshToken: got %q, want %q", got.RefreshToken, tokens.RefreshToken)
	}
	if got.ExpiresIn != tokens.ExpiresIn {
		t.Fatalf("ExpiresIn: got %d, want %d", got.ExpiresIn, tokens.ExpiresIn)
	}
}

func TestAuthCodeStore_ExchangeDeletesCode(t *testing.T) {
	store := NewAuthCodeStore()
	defer store.Stop()

	code, err := store.Store(testTokenPair())
	if err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	// First exchange succeeds.
	if _, err := store.Exchange(code); err != nil {
		t.Fatalf("first Exchange() error: %v", err)
	}

	// Second exchange fails — code already consumed.
	if _, err := store.Exchange(code); err == nil {
		t.Fatal("expected error on second Exchange(), got nil")
	}
}

func TestAuthCodeStore_InvalidCodeReturnsError(t *testing.T) {
	store := NewAuthCodeStore()
	defer store.Stop()

	_, err := store.Exchange("nonexistent-code")
	if err == nil {
		t.Fatal("expected error for invalid code, got nil")
	}
	if !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("expected ErrCodeInvalid, got %v", err)
	}
}

func TestAuthCodeStore_ExpiredCodeReturnsError(t *testing.T) {
	store := NewAuthCodeStore()
	defer store.Stop()

	tokens := testTokenPair()
	code, err := store.Store(tokens)
	if err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	// Manually expire the code.
	store.mu.Lock()
	store.codes[code].expiresAt = time.Now().Add(-1 * time.Second)
	store.mu.Unlock()

	_, err = store.Exchange(code)
	if err == nil {
		t.Fatal("expected error for expired code, got nil")
	}
	if !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("expected ErrCodeInvalid, got %v", err)
	}

	// Verify the expired code was also deleted.
	store.mu.Lock()
	_, exists := store.codes[code]
	store.mu.Unlock()
	if exists {
		t.Fatal("expired code should have been deleted after exchange attempt")
	}
}
