package infra_test

// Unit tests for the in-process, board-scoped SSE hub (T4 + boards BRD-05):
//   - Broadcast(boardID) delivers board.state_changed.v1 (events.md shape) to
//     every connection registered under that board — team-editor and
//     public-viewer alike — and to no other board's connections.
//   - CloseToken(boardID, token) closes exactly the connections registered
//     under that board's token, leaving every other connection (other tokens,
//     team-editor, other boards) open.
//   - register/unregister/broadcast is safe under `go test -race` from
//     multiple goroutines.

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// TestHub_Broadcast_DeliversToEveryRegisteredConnection covers T4's first DoD
// line: a single Broadcast() call must reach every live connection of the
// broadcast board — the team-editor connection (no token) and every
// public-viewer connection (registered under a token) — with the events.md
// v1 shape intact.
func TestHub_Broadcast_DeliversToEveryRegisteredConnection(t *testing.T) {
	h := infra.NewHub()
	boardID := uuid.Must(uuid.NewV7())

	_, teamCh := h.Register(boardID, "") // team-editor: no token
	_, viewerACh := h.Register(boardID, "token-a")
	_, viewerBCh := h.Register(boardID, "token-b")

	evt := ports.Event{
		EventID:    "01960000-0000-7000-8000-000000000001",
		EventType:  "board.state_changed",
		Version:    1,
		OccurredAt: time.Now().UTC(),
	}

	h.Broadcast(boardID, evt)

	for label, ch := range map[string]<-chan ports.Event{
		"team-editor connection":  teamCh,
		"public viewer (token-a)": viewerACh,
		"public viewer (token-b)": viewerBCh,
	} {
		select {
		case got, ok := <-ch:
			require.Truef(t, ok, "%s: channel closed before delivering the broadcast event", label)
			require.Equalf(t, evt, got, "%s: delivered event does not match the broadcast event", label)
		case <-time.After(time.Second):
			t.Fatalf("%s: did not receive the broadcast event within 1s", label)
		}
	}
}

// TestHub_Broadcast_ScopedToOneBoard pins boards BRD-05: a broadcast for
// board A must never reach board B's connections — neither its team-editor
// connection nor its viewers.
func TestHub_Broadcast_ScopedToOneBoard(t *testing.T) {
	h := infra.NewHub()
	boardA := uuid.Must(uuid.NewV7())
	boardB := uuid.Must(uuid.NewV7())

	_, aTeamCh := h.Register(boardA, "")
	_, bTeamCh := h.Register(boardB, "")
	_, bViewerCh := h.Register(boardB, "token-b")

	h.Broadcast(boardA, ports.Event{
		EventID:    "evt-a",
		EventType:  "board.state_changed",
		Version:    1,
		OccurredAt: time.Now().UTC(),
	})

	select {
	case _, ok := <-aTeamCh:
		require.True(t, ok, "board A team-editor: channel closed before delivering the broadcast")
	case <-time.After(time.Second):
		t.Fatal("board A team-editor: did not receive its own board's broadcast within 1s")
	}

	requireOpenAndIdle(t, bTeamCh, "board B team-editor connection")
	requireOpenAndIdle(t, bViewerCh, "board B viewer connection")
}

// TestHub_CloseToken_ClosesExactlyThatTokensConnections covers T4's second
// DoD line and is what makes T9/T12's AC-11 revoke-closes-SSE test possible:
// CloseToken(boardID, token) must close every connection registered under
// that board's token and must NOT touch connections under a different token,
// the team-editor connection, or another board's connections.
func TestHub_CloseToken_ClosesExactlyThatTokensConnections(t *testing.T) {
	h := infra.NewHub()
	boardID := uuid.Must(uuid.NewV7())
	otherBoardID := uuid.Must(uuid.NewV7())

	_, tokenAConn1 := h.Register(boardID, "token-a")
	_, tokenAConn2 := h.Register(boardID, "token-a")
	_, tokenBConn := h.Register(boardID, "token-b")
	_, teamConn := h.Register(boardID, "")
	_, otherBoardConn := h.Register(otherBoardID, "token-c")

	h.CloseToken(boardID, "token-a")

	requireClosed(t, tokenAConn1, "token-a connection #1")
	requireClosed(t, tokenAConn2, "token-a connection #2")
	requireOpenAndIdle(t, tokenBConn, "token-b connection")
	requireOpenAndIdle(t, teamConn, "team-editor connection")
	requireOpenAndIdle(t, otherBoardConn, "other board's viewer connection")
}

// TestHub_ConcurrentRegisterBroadcastUnregister_NoRace exercises the
// concurrent-safety DoD line: register/unregister/broadcast from multiple
// goroutines simultaneously must not race (run with `go test -race`).
func TestHub_ConcurrentRegisterBroadcastUnregister_NoRace(t *testing.T) {
	h := infra.NewHub()
	boardID := uuid.Must(uuid.NewV7())

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(n int) {
			defer wg.Done()
			id, ch := h.Register(boardID, "token-race")
			h.Broadcast(boardID, ports.Event{
				EventID:    "evt",
				EventType:  "board.state_changed",
				Version:    1,
				OccurredAt: time.Now().UTC(),
			})
			// Drain best-effort so Broadcast's send doesn't block forever on
			// an unbuffered channel; ignore the value, this test targets the
			// race detector, not delivery ordering.
			select {
			case <-ch:
			case <-time.After(100 * time.Millisecond):
			}
			h.Unregister(boardID, "token-race", id)
		}(i)
	}

	wg.Wait()
	h.CloseToken(boardID, "token-race")
}

func requireClosed(t *testing.T, ch <-chan ports.Event, label string) {
	t.Helper()
	select {
	case _, ok := <-ch:
		require.Falsef(t, ok, "%s: expected the channel to be closed by CloseToken, but it was still open", label)
	case <-time.After(time.Second):
		t.Fatalf("%s: expected the channel to be closed by CloseToken, but the read timed out", label)
	}
}

func requireOpenAndIdle(t *testing.T, ch <-chan ports.Event, label string) {
	t.Helper()
	select {
	case _, ok := <-ch:
		require.Truef(t, ok, "%s: expected the channel to remain open, but it was closed", label)
		t.Fatalf("%s: unexpectedly received a value on an idle connection", label)
	default:
		// No value pending and not closed — correct: this connection was
		// never touched.
	}
}
