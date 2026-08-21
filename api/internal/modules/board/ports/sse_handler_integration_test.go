//go:build integration

package ports_test

// RED for T9 (docs/features/board/tasks/T9-ports-sse-endpoints.md):
// api/internal/modules/board/ports/sse_handler.go does not exist yet —
// ports.NewSSEHandler is expected to land there, wiring the T4 infra.Hub and
// app.StateService behind GET /api/v1/boards/{boardId}/events (team-editor) and
// GET /api/v1/public/{token}/events (public viewer, contracts/openapi.yaml
// operationIds streamBoardEvents/streamPublicBoardEvents).
//
// Covers T9's DoD, all under AC-11:
//   - a public SSE request with an invalid/revoked token returns 404
//     board.link_invalid before registering any connection
//   - a valid public SSE connection receives a board.state_changed event
//     (events.md v1 shape) when the hub broadcasts
//   - revoking the active public link (app.LinkService.RevokePublicLink, T6,
//     already implemented) closes an already-open SSE connection registered
//     under that token synchronously — the HTTP response stream ends on its
//     own, not merely once the client gives up
//
// Deliberately self-contained: the public link under test is issued/revoked
// directly through the already-implemented app.LinkService (T6) rather than
// through T8's HTTP handler, so this RED does not depend on sibling
// in-flight tasks (T8/T10) for its own compile/run status. Type/helper names
// below are prefixed t9* to avoid clashing with them.
//
// No production code exists yet for ports.NewSSEHandler — this is the RED
// step; the package will not compile until T9 adds it.

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/app"
	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/database/dbtest"
	"github.com/genkovich/task-tracker/api/migrations"
)

// seeded board from migration 000007 (data-model.md, CONTEXT.md: exactly one
// board row) — matches infra's postgres_repo_integration_test.go.
var t9SeedBoardID = uuid.MustParse("019a0000-0000-7000-8000-000000000101")

// t9WireErrorResponse mirrors the Error envelope (contracts/openapi.yaml).
type t9WireErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// t9SSEEvent mirrors the board.state_changed.v1 shape (contracts/events.md).
// WireEventName is not JSON — it carries the value of the SSE "event:" line
// preceding "data:", i.e. the name browsers dispatch to
// EventSource.addEventListener (review 2026-08-21, root B pin).
type t9SSEEvent struct {
	WireEventName string `json:"-"`
	EventID       string `json:"event_id"`
	EventType     string `json:"event_type"`
	Version       int    `json:"version"`
	OccurredAt    string `json:"occurred_at"`
}

// t9SSEFixture bundles a running server exposing only the SSE routes (T9)
// plus direct access to the real hub/link service, so tests can trigger a
// broadcast or a revoke exactly as app/T6 already does — without depending
// on T7/T8's own HTTP routes, which are separate in-flight RED work.
type t9SSEFixture struct {
	ts      *httptest.Server
	hub     *infra.Hub
	linkSvc *app.LinkService
}

func setupT9SSEServer(t *testing.T) t9SSEFixture {
	t.Helper()

	ctx := context.Background()
	c := dbtest.StartPostgres(ctx, t)

	if err := database.RunMigrations(migrations.FS, c.DSN); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := infra.NewPostgresRepository(c.DB)
	hub := infra.NewHub()

	stateSvc := app.NewStateService(repo)
	linkSvc := app.NewLinkService(repo, hub)

	handler := ports.NewSSEHandler(hub, stateSvc, stateSvc)

	r := chi.NewRouter()
	r.Route("/api/v1", handler.RegisterRoutes)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return t9SSEFixture{ts: ts, hub: hub, linkSvc: linkSvc}
}

// t9OpenSSEStream issues a GET request to path and returns the live response
// plus a channel of decoded events read off the stream as they arrive. The
// events channel and the returned done channel are both closed when the
// stream ends.
func t9OpenSSEStream(t *testing.T, ts *httptest.Server, path string) (*http.Response, <-chan t9SSEEvent, <-chan struct{}) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // caller closes resp.Body
	require.NoError(t, err)

	events := make(chan t9SSEEvent, 8)
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer close(events)

		scanner := bufio.NewScanner(resp.Body)
		var eventName string
		for scanner.Scan() {
			line := scanner.Text()
			if name, ok := strings.CutPrefix(line, "event:"); ok {
				eventName = strings.TrimSpace(name)
				continue
			}
			data, ok := strings.CutPrefix(line, "data:")
			if !ok {
				continue
			}
			var evt t9SSEEvent
			if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &evt); err == nil {
				evt.WireEventName = eventName
				events <- evt
			}
			eventName = ""
		}
	}()

	return resp, events, done
}

// TestSSEHandler_StreamPublicBoardEvents_InvalidToken_Returns404LinkInvalid
// covers T9's first DoD line and AC-11's "revoked/unknown token" branch: a
// request for a token with no active public link must be rejected with 404
// board.link_invalid before any SSE connection is registered.
func TestSSEHandler_StreamPublicBoardEvents_InvalidToken_Returns404LinkInvalid(t *testing.T) {
	f := setupT9SSEServer(t)

	resp, err := http.Get(f.ts.URL + "/api/v1/public/revoked-or-unknown-token/events") //nolint:noctx // test helper
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equalf(t, http.StatusNotFound, resp.StatusCode,
		"GET /api/v1/public/{token}/events (invalid token) status")

	var body t9WireErrorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "board.link_invalid", body.Error.Code,
		"AC-11: unknown/revoked token must map to board.link_invalid")
}

// TestSSEHandler_StreamPublicBoardEvents_ValidToken_ReceivesBroadcastEvent
// covers T9's second DoD line: a viewer holding a valid token, once
// connected, receives a board.state_changed.v1 event when the hub
// broadcasts.
func TestSSEHandler_StreamPublicBoardEvents_ValidToken_ReceivesBroadcastEvent(t *testing.T) {
	f := setupT9SSEServer(t)
	ctx := context.Background()

	link, err := f.linkSvc.IssuePublicLink(ctx, t9SeedBoardID)
	require.NoError(t, err)

	resp, events, done := t9OpenSSEStream(t, f.ts, "/api/v1/public/"+link.Token+"/events")
	defer resp.Body.Close()
	require.Equalf(t, http.StatusOK, resp.StatusCode, "GET /api/v1/public/{token}/events (valid token) status")

	// There is no synchronous "connection registered" signal to wait on from
	// outside the handler, so broadcast repeatedly until the event shows up
	// or we give up.
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	var got t9SSEEvent
	var received bool
	for !received {
		select {
		case evt := <-events:
			got = evt
			received = true
		case <-tick.C:
			f.hub.Broadcast(t9SeedBoardID, ports.Event{
				EventID:    "01960000-0000-7000-8000-000000000099",
				EventType:  "board.state_changed",
				Version:    1,
				OccurredAt: time.Now().UTC(),
			})
		case <-deadline:
			t.Fatal("streamPublicBoardEvents never delivered a board.state_changed event after repeated broadcasts")
		case <-done:
			t.Fatal("SSE stream closed unexpectedly before delivering any event")
		}
	}

	require.Equal(t, "board.state_changed", got.WireEventName,
		"root B pin: the message must carry the SSE 'event:' line clients subscribe to via addEventListener")
	require.Equal(t, "board.state_changed", got.EventType, "events.md v1 shape: event_type")
	require.Equal(t, 1, got.Version, "events.md v1 shape: version")
	require.NotEmpty(t, got.EventID, "events.md v1 shape: event_id")
}

// TestSSEHandler_RevokePublicLink_ClosesActiveConnectionSynchronously covers
// T9's third DoD line and AC-11's cross-context scenario: once
// app.LinkService.RevokePublicLink (T6, already implemented) revokes the
// token a viewer is connected with, that viewer's SSE HTTP response stream
// must end on its own — synchronously with the revoke, not because the
// client gave up or timed out.
func TestSSEHandler_RevokePublicLink_ClosesActiveConnectionSynchronously(t *testing.T) {
	f := setupT9SSEServer(t)
	ctx := context.Background()

	link, err := f.linkSvc.IssuePublicLink(ctx, t9SeedBoardID)
	require.NoError(t, err)

	resp, _, done := t9OpenSSEStream(t, f.ts, "/api/v1/public/"+link.Token+"/events")
	defer resp.Body.Close()
	require.Equalf(t, http.StatusOK, resp.StatusCode, "GET /api/v1/public/{token}/events (valid token) status")

	// Give the handler time to register the connection with the hub before
	// revoking — there is no synchronous "registered" signal to wait on.
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, f.linkSvc.RevokePublicLink(ctx, t9SeedBoardID))

	select {
	case <-done:
		// The SSE stream ended on its own — the revoke closed this
		// connection synchronously (AC-11).
	case <-time.After(2 * time.Second):
		t.Fatal("SSE stream for a revoked token's connection did not close within 2s — AC-11 requires the revoke to close it synchronously")
	}
}
