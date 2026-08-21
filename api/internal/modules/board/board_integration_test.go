//go:build integration

package board_test

// Пінінг рев'ю 2026-08-21, корінь A: board раніше монтувався на root router
// повз server.New і тому обходив rate limiter і metrics middleware, а SSE
// різався per-request timeout'ом. Ці тести йдуть через справжній server.New
// з board.New(db) у opts — точнісінько як cmd/api/main.go — і пінять:
//   - board-роути видимі HTTP-метрикам (http_requests_total по route-патерну)
//   - board-роути під загальним rate limit 60/хв (61-й запит → 429)
//   - SSE-стрім живе довше за per-request timeout і після нього все ще
//     доставляє події (streaming-група без middleware.Timeout)

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/internal/server"
	"github.com/genkovich/task-tracker/api/migrations"
)

// setupBoardServer wires board through server.New exactly the way
// cmd/api/main.go does (no root-router side mount), with opts appended.
func setupBoardServer(t *testing.T, opts ...any) *httptest.Server {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	serverOpts := append([]any{board.New(c.DB)}, opts...)
	s := server.New(c.DB, "*", nil, serverOpts...)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestBoardWiring_MetricsAndRateLimit pins two halves of root A: board
// routes must pass through the server's metrics middleware (the review found
// zero series) and through the general 60/min rate limit (the review proved
// 81×200 in a row).
func TestBoardWiring_MetricsAndRateLimit(t *testing.T) {
	ts := setupBoardServer(t)

	// 60 requests fit the budget; the 61st within the same minute must be
	// throttled — before the fix board bypassed the limiter entirely.
	for i := range 60 {
		resp, err := http.Get(ts.URL + "/api/v1/board") //nolint:noctx // test helper
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		require.Equalf(t, http.StatusOK, resp.StatusCode, "GET /api/v1/board #%d within the 60/min budget", i+1)
	}

	resp, err := http.Get(ts.URL + "/api/v1/board") //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
		"the 61st GET /api/v1/board within a minute must hit the general rate limit")

	// /metrics is mounted outside the limiter, so this scrape always works.
	metricsResp, err := http.Get(ts.URL + "/metrics") //nolint:noctx // test helper
	require.NoError(t, err)
	defer metricsResp.Body.Close()
	require.Equal(t, http.StatusOK, metricsResp.StatusCode)

	raw, err := io.ReadAll(metricsResp.Body)
	require.NoError(t, err)
	require.Contains(t, string(raw), `http_requests_total{method="GET",route="/api/v1/board"`,
		"board requests must be recorded by the HTTP metrics middleware")
}

// TestBoardWiring_SSEOutlivesRequestTimeout pins the streaming half of root
// A: with a deliberately short per-request timeout the SSE stream must stay
// open past it and still deliver events — proof the SSE routes live outside
// middleware.Timeout while ordinary routes keep it.
func TestBoardWiring_SSEOutlivesRequestTimeout(t *testing.T) {
	const requestTimeout = 500 * time.Millisecond
	ts := setupBoardServer(t, server.WithRequestTimeout(requestTimeout))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/api/v1/board/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /api/v1/board/events status")

	lines := make(chan string, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	// Outlive the request timeout threefold before provoking any event: a
	// stream still under middleware.Timeout would be cancelled here.
	select {
	case <-done:
		t.Fatalf("SSE stream closed within %v — the per-request timeout must not apply to streaming routes", 3*requestTimeout)
	case <-time.After(3 * requestTimeout):
	}

	// A mutation broadcasts board.state_changed; the long-lived stream must
	// still be subscribed and deliver it.
	body, err := json.Marshal(map[string]any{"title": "outlive the timeout"})
	require.NoError(t, err)
	createResp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body)) //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, createResp.Body)
	createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-lines:
			if strings.HasPrefix(line, "data:") && strings.Contains(line, "board.state_changed") {
				return // stream alive well past the timeout and still delivering
			}
		case <-done:
			t.Fatal("SSE stream closed before delivering the broadcast event")
		case <-deadline:
			t.Fatal("no board.state_changed arrived on a stream older than the request timeout")
		}
	}
}

// TestPublicRoutes_MutatingMethodsRefused covers AC-10 against the fully
// wired server (review 2026-08-21 re2, #10): the earlier version ran on a
// fixture that registered only the public handler, so its 404s were
// guaranteed by the empty router — a tautology. Here server.New carries
// every board route (task CRUD included), the GET sanity check proves the
// public path itself is live on this router, and every mutating method
// under /api/v1/public/{token}/... is still refused by routing (405 on the
// read-only board path, 404 on paths that are not registered), never
// executed — even with a VALID token.
func TestPublicRoutes_MutatingMethodsRefused(t *testing.T) {
	ts := setupBoardServer(t)
	ctx := context.Background()

	// Issue a real public link over the team-editor HTTP route.
	linkResp, err := http.Post(ts.URL+"/api/v1/board/public-link", "application/json", nil) //nolint:noctx // test helper
	require.NoError(t, err)
	var link struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(linkResp.Body).Decode(&link))
	linkResp.Body.Close()
	require.Equal(t, http.StatusCreated, linkResp.StatusCode)
	require.NotEmpty(t, link.Token)

	base := ts.URL + "/api/v1/public/" + link.Token

	// Sanity: the read-only route answers on this very server, so the
	// refusals below prove routing policy, not an unregistered path.
	getResp, err := http.Get(base + "/board") //nolint:noctx // test helper
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, getResp.Body)
	getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode, "GET public board must work with a valid token")

	cases := []struct {
		method string
		url    string
	}{
		{http.MethodPost, base + "/board"},
		{http.MethodPatch, base + "/board"},
		{http.MethodDelete, base + "/board"},
		{http.MethodPost, base + "/tasks"},
		{http.MethodDelete, base + "/tasks/" + uuid.NewString()},
	}

	for _, tc := range cases {
		req, err := http.NewRequestWithContext(ctx, tc.method, tc.url, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		require.Containsf(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode,
			"AC-10: %s %s must be refused by routing, got %d", tc.method, tc.url, resp.StatusCode)
	}
}
