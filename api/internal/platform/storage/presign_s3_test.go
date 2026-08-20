package storage_test

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Storage_PresignPut_ReturnsURLWithBoundedTTL(t *testing.T) {
	// Presigning is a local computation (no network round-trip), so an
	// unstarted/never-hit handler is fine here.
	s, _ := newTestS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("PresignPut must not perform an HTTP request, got %s %s", r.Method, r.URL.Path)
	}))

	const key = "submissions/v1/report.pdf"
	const ttl = 15 * time.Minute

	got, err := s.PresignPut(context.Background(), key, "application/pdf", 2048, ttl)

	require.NoError(t, err)
	require.NotEmpty(t, got)

	u, err := url.Parse(got)
	require.NoError(t, err, "PresignPut must return a valid URL")
	assert.Contains(t, u.Path, "test-bucket/"+key, "presigned URL must target the requested bucket/key")

	expires := u.Query().Get("X-Amz-Expires")
	require.NotEmpty(t, expires, "presigned URL must carry an explicit expiry (X-Amz-Expires)")
	gotTTL, convErr := strconv.Atoi(expires)
	require.NoError(t, convErr)
	assert.Equal(t, int(ttl.Seconds()), gotTTL, "presigned URL TTL must match the requested duration")
}

func TestS3Storage_PresignGet_ReturnsURLWithBoundedTTL(t *testing.T) {
	s, _ := newTestS3(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("PresignGet must not perform an HTTP request, got %s %s", r.Method, r.URL.Path)
	}))

	const key = "submissions/v1/report.pdf"
	const ttl = 10 * time.Minute

	got, err := s.PresignGet(context.Background(), key, ttl)

	require.NoError(t, err)
	require.NotEmpty(t, got)

	u, err := url.Parse(got)
	require.NoError(t, err, "PresignGet must return a valid URL")
	assert.Contains(t, u.Path, "test-bucket/"+key, "presigned URL must target the requested bucket/key")

	expires := u.Query().Get("X-Amz-Expires")
	require.NotEmpty(t, expires, "presigned URL must carry an explicit expiry (X-Amz-Expires)")
	gotTTL, convErr := strconv.Atoi(expires)
	require.NoError(t, convErr)
	assert.Equal(t, int(ttl.Seconds()), gotTTL, "presigned URL TTL must match the requested duration")
}

func TestS3Storage_Head_ExistingObject_ReturnsSizeAndExists(t *testing.T) {
	const key = "avatars/head-me.png"
	const wantSize = 4096

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.Header().Set("Content-Length", strconv.Itoa(wantSize))
		w.WriteHeader(http.StatusOK)
	})

	s, _ := newTestS3(t, handler)

	size, exists, err := s.Head(context.Background(), key)

	require.NoError(t, err)
	assert.True(t, exists, "Head must report an existing object as present")
	assert.Equal(t, int64(wantSize), size, "Head must report the object's byte size")
}

func TestS3Storage_Head_MissingObject_ReturnsNotExists(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.WriteHeader(http.StatusNotFound)
	})

	s, _ := newTestS3(t, handler)

	size, exists, err := s.Head(context.Background(), "avatars/never-uploaded.png")

	require.NoError(t, err, "a missing object is not an error condition for Head")
	assert.False(t, exists, "Head must report a missing object as absent")
	assert.Equal(t, int64(0), size)
}
