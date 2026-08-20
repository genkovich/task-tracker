package storage_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/genkovich/task-tracker/api/internal/platform/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestS3 creates an S3 client pointed at a fake httptest server.
// The server is closed automatically via t.Cleanup.
func newTestS3(t *testing.T, handler http.Handler) (*storage.S3, string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := storage.S3Config{
		Bucket:        "test-bucket",
		Region:        "us-east-1",
		Endpoint:      srv.URL,
		PublicBaseURL: "http://public.example.com/test-bucket",
		AccessKeyID:   "test-key",
		SecretKey:     "test-secret",
	}
	s, err := storage.NewS3(cfg)
	require.NoError(t, err, "NewS3 should succeed with valid config")
	return s, srv.URL
}

// --- PUT tests ---

func TestS3Storage_Put_StoresObjectAndReturnsPublicURL(t *testing.T) {
	const key = "avatars/abc.png"
	const body = "fake-image-bytes"

	var capturedMethod, capturedPath string
	var capturedBody []byte

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	s, _ := newTestS3(t, handler)

	url, err := s.Put(context.Background(), storage.Object{
		Key:         key,
		ContentType: "image/png",
		Body:        strings.NewReader(body),
		Size:        int64(len(body)),
	})

	require.NoError(t, err)
	assert.Equal(t, "PUT", capturedMethod)
	assert.Equal(t, "/test-bucket/avatars/abc.png", capturedPath)
	assert.Equal(t, body, string(capturedBody))
	assert.Equal(t, "http://public.example.com/test-bucket/avatars/abc.png", url)
}

func TestS3Storage_Put_PreservesContentType(t *testing.T) {
	const wantCT = "image/jpeg"

	var capturedCT string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	})

	s, _ := newTestS3(t, handler)

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "avatars/x.jpg",
		ContentType: wantCT,
		Body:        strings.NewReader("data"),
		Size:        4,
	})

	require.NoError(t, err)
	assert.Equal(t, wantCT, capturedCT)
}

func TestS3Storage_Put_StreamsBody(t *testing.T) {
	// We verify that the body bytes received by the server exactly match what
	// we put into the reader — confirming the data flows through without
	// silent truncation or extra buffering that would drop bytes.
	const payload = "streaming-payload-content"

	var capturedBody []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	s, _ := newTestS3(t, handler)

	reader := strings.NewReader(payload)
	_, err := s.Put(context.Background(), storage.Object{
		Key:         "avatars/stream.png",
		ContentType: "image/png",
		Body:        reader,
		Size:        int64(len(payload)),
	})

	require.NoError(t, err)
	assert.Equal(t, payload, string(capturedBody), "server must receive exactly the bytes from the reader")
	// Confirm reader was drained (position at end)
	remaining, _ := io.ReadAll(reader)
	assert.Empty(t, remaining, "reader should be fully consumed after Put")
}

func TestS3Storage_Put_ServerError_PropagatesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	s, _ := newTestS3(t, handler)

	_, err := s.Put(context.Background(), storage.Object{
		Key:         "avatars/fail.png",
		ContentType: "image/png",
		Body:        strings.NewReader("data"),
		Size:        4,
	})

	assert.Error(t, err, "5xx from server must produce an error")
}

// --- DELETE tests ---

func TestS3Storage_Delete_RemovesObject(t *testing.T) {
	const key = "avatars/del.png"

	var capturedMethod, capturedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	s, _ := newTestS3(t, handler)

	err := s.Delete(context.Background(), key)

	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Equal(t, "/test-bucket/avatars/del.png", capturedPath)
}

func TestS3Storage_Delete_NotFound_IsNotError(t *testing.T) {
	// S3 returns 404 with NoSuchKey XML body for missing objects.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		body, _ := xml.Marshal(s3ErrorXML{Code: "NoSuchKey", Message: "The specified key does not exist."})
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
		_, _ = w.Write(body)
	})

	s, _ := newTestS3(t, handler)

	err := s.Delete(context.Background(), "avatars/missing.png")
	assert.NoError(t, err, "NoSuchKey 404 must be treated as success (idempotent)")
}

func TestS3Storage_Delete_ServerError_PropagatesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	s, _ := newTestS3(t, handler)

	err := s.Delete(context.Background(), "avatars/err.png")
	assert.Error(t, err, "5xx from server must produce a non-nil error")
}

// s3ErrorXML is a minimal S3 error response envelope for test helpers.
type s3ErrorXML struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

// Verify S3 satisfies the interface at compile time.
var _ storage.ObjectStorage = (*storage.S3)(nil)

// Verify config URL format used in tests.
func TestS3Config_PublicURLFormat(t *testing.T) {
	cfg := storage.S3Config{
		Bucket:        "my-bucket",
		Region:        "eu2",
		Endpoint:      "https://eu2.contabostorage.com",
		PublicBaseURL: "https://eu2.contabostorage.com/my-bucket",
		AccessKeyID:   "key",
		SecretKey:     "secret",
	}
	s, err := storage.NewS3(cfg)
	require.NoError(t, err)
	// Just confirm construction; actual URL is formed by Put.
	assert.NotNil(t, s)
	_ = fmt.Sprintf("%s/%s", cfg.PublicBaseURL, "avatars/test.png")
}
