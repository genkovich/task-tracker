package storage_test

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenFromURL extracts the "token" query parameter from a presigned local
// URL so tests can feed it back into ValidateToken.
func tokenFromURL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err, "presigned URL must be a valid URL")
	tok := u.Query().Get("token")
	require.NotEmpty(t, tok, "presigned URL must carry a token query parameter")
	return tok
}

func TestLocalStorage_PresignPut_ReturnsURLWithKeyAndToken(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const key = "submissions/v1/report.pdf"
	got, err := s.PresignPut(context.Background(), key, "application/pdf", 2048, 15*time.Minute)

	require.NoError(t, err)
	assert.NotEmpty(t, got, "PresignPut must return a non-empty URL")
	assert.Contains(t, got, key, "presigned URL must reference the object key, got %q", got)
	_ = tokenFromURL(t, got)
}

func TestLocalStorage_PresignGet_ReturnsURLWithKeyAndToken(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const key = "submissions/v1/report.pdf"
	got, err := s.PresignGet(context.Background(), key, 15*time.Minute)

	require.NoError(t, err)
	assert.NotEmpty(t, got, "PresignGet must return a non-empty URL")
	assert.Contains(t, got, key, "presigned URL must reference the object key, got %q", got)
	_ = tokenFromURL(t, got)
}

func TestLocalStorage_ValidateToken_AcceptsValidUnexpiredToken(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const key = "submissions/v1/report.pdf"
	presigned, err := s.PresignPut(context.Background(), key, "application/pdf", 2048, 15*time.Minute)
	require.NoError(t, err)
	token := tokenFromURL(t, presigned)

	now := time.Now() // within the 15-minute TTL window
	err = s.ValidateToken(key, token, now)

	assert.NoError(t, err, "a fresh, matching-key token must be accepted")
}

func TestLocalStorage_ValidateToken_RejectsExpiredToken(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const key = "submissions/v1/report.pdf"
	presigned, err := s.PresignPut(context.Background(), key, "application/pdf", 2048, time.Minute)
	require.NoError(t, err)
	token := tokenFromURL(t, presigned)

	// spec §6.1: "a link alone is never sufficient proof of access" — an
	// expired token must be rejected even though the signature is valid.
	afterExpiry := time.Now().Add(2 * time.Minute)
	err = s.ValidateToken(key, token, afterExpiry)

	require.Error(t, err, "an expired token must be rejected")
	assert.Contains(t, strings.ToLower(err.Error()), "expired", "error must explain the token expired, got %q", err.Error())
}

func TestLocalStorage_ValidateToken_RejectsTokenWithSubstitutedKey(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const originalKey = "submissions/v1/mine.pdf"
	const otherKey = "submissions/v1/someone-elses.pdf"

	presigned, err := s.PresignPut(context.Background(), originalKey, "application/pdf", 2048, 15*time.Minute)
	require.NoError(t, err)
	token := tokenFromURL(t, presigned)

	// spec §6.1: reusing a token issued for one key against a different key
	// must be refused — a token alone is not proof of access to any key.
	err = s.ValidateToken(otherKey, token, time.Now())

	require.Error(t, err, "a token issued for a different key must be rejected")
}

func TestLocalStorage_ValidateToken_RejectsGarbageToken(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	err := s.ValidateToken("submissions/v1/report.pdf", "not-a-real-token", time.Now())

	assert.Error(t, err, "a malformed token must be rejected")
}

func TestLocalStorage_Head_ExistingObject_ReturnsSizeAndExists(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	const key = "avatars/head-me.png"
	const content = "twelve-bytes"
	_, err := s.Put(context.Background(), storage.Object{
		Key:         key,
		ContentType: "image/png",
		Body:        strings.NewReader(content),
		Size:        int64(len(content)),
	})
	require.NoError(t, err)

	size, exists, err := s.Head(context.Background(), key)

	require.NoError(t, err)
	assert.True(t, exists, "Head must report an existing object as present")
	assert.Equal(t, int64(len(content)), size, "Head must report the exact byte size on disk")
}

func TestLocalStorage_Head_MissingObject_ReturnsNotExists(t *testing.T) {
	baseDir := t.TempDir()
	s := storage.NewLocal(baseDir, "http://localhost/files", storage.WithSigningSecret("test-secret"))

	size, exists, err := s.Head(context.Background(), "avatars/never-uploaded.png")

	require.NoError(t, err, "a missing object is not an error condition for Head")
	assert.False(t, exists, "Head must report a missing object as absent")
	assert.Equal(t, int64(0), size)
}
