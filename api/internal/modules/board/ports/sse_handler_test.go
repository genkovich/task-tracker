package ports

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/domain"
)

// TestWriteSSEEvent_NamedEvent pins the exact wire format of one SSE message
// (review 2026-08-21, root B): the client subscribes to the NAMED event
// "board.state_changed" via EventSource.addEventListener, so a message
// without the "event:" line is silently dropped by every browser — live
// push would never work while both sides' isolated tests stayed green.
func TestWriteSSEEvent_NamedEvent(t *testing.T) {
	rec := httptest.NewRecorder()

	writeSSEEvent(rec, Event{
		EventID:    "01960000-0000-7000-8000-000000000042",
		EventType:  "board.state_changed",
		Version:    1,
		OccurredAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
	})

	body := rec.Body.String()
	lines := strings.Split(body, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "an SSE message is at least an event: and a data: line")

	require.Equal(t, "event: board.state_changed", lines[0],
		"the first line must name the event exactly as clients subscribe to it")
	require.True(t, strings.HasPrefix(lines[1], "data: "), "the data: line follows the event: line")
	require.Contains(t, lines[1], `"event_type":"board.state_changed"`)
	require.True(t, strings.HasSuffix(body, "\n\n"), "an SSE message ends with a blank line")
}

// Review 2026-08-21 re2, #8: token validation and hub registration are not
// atomic — a revoke (CloseToken) landing between them finds no bucket to
// close, so the freshly registered connection would live forever (the
// heartbeat keeps it up and no later CloseToken ever sees it). The handler
// must re-check the token after registering and tear the stream down when
// the link is already revoked. The interleaving modelled here: validation
// ok → revoke → register → re-check.
func TestStreamPublicBoardEvents_RevokeBetweenValidateAndRegister_ClosesStream(t *testing.T) {
	registry := &recordingRegistry{events: make(chan Event)}
	svc := &revokedAfterFirstCheckStateService{}

	h := NewSSEHandler(registry, svc, &neverCalledBoardStateService{t: t})
	r := chi.NewRouter()
	h.RegisterPublicRoutes(r)

	// The timeout bounds the failure mode, not the happy path: an unfixed
	// handler streams until the client context ends.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/public/some-token/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code,
		"a token revoked between validation and registration must be rejected, not streamed")
	require.Contains(t, rec.Body.String(), "board.link_invalid")
	require.True(t, registry.unregistered, "the raced connection must be unregistered")
}

// revokedAfterFirstCheckStateService models the racing revoke: the first
// validation succeeds and the revoke lands right after it — every later
// check sees the link gone.
type revokedAfterFirstCheckStateService struct {
	calls int
}

func (s *revokedAfterFirstCheckStateService) GetPublicBoardState(_ context.Context, _ string) (*PublicBoardState, error) {
	s.calls++
	if s.calls == 1 {
		return &PublicBoardState{}, nil
	}
	return nil, domain.ErrLinkNotFound
}

// recordingRegistry satisfies SSERegistry and records whether the
// connection it handed out was unregistered.
type recordingRegistry struct {
	events       chan Event
	unregistered bool
}

func (r *recordingRegistry) Subscribe(uuid.UUID, string) (<-chan Event, func()) {
	return r.events, func() { r.unregistered = true }
}

// neverCalledBoardStateService satisfies SSEBoardStateService for the
// public-viewer path, which must never consult the team-editor board lookup.
type neverCalledBoardStateService struct {
	t *testing.T
}

func (s *neverCalledBoardStateService) GetBoardState(context.Context, uuid.UUID) (*BoardState, error) {
	s.t.Fatal("the public-viewer stream must not consult the team-editor board state service")
	return nil, nil
}
