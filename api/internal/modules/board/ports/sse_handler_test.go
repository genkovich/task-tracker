package ports

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
