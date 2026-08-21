package infra_test

// Unit tests for the in-process SSE hub (T4). Per T4's DoD:
//   - Broadcast() delivers board.state_changed.v1 (events.md shape) to every
//     registered connection, team-editor and public-viewer alike.
//   - CloseToken(token) closes exactly the connections registered under that
//     token, leaving every other connection (other tokens, team-editor) open.
//   - register/unregister/broadcast is safe under `go test -race` from
//     multiple goroutines.
//
// No production code exists yet for infra.Hub / ports.Event / ports.Broadcaster
// — this is the RED step.

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/board/infra"
	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// TestHub_Broadcast_DeliversToEveryRegisteredConnection covers T4's first DoD
// line: a single Broadcast() call must reach every live connection — the
// team-editor connection (no token) and every public-viewer connection
// (registered under a token) — with the events.md v1 shape intact.
func TestHub_Broadcast_DeliversToEveryRegisteredConnection(t *testing.T) {
	h := infra.NewHub()

	_, teamCh := h.Register("") // team-editor: no token
	_, viewerACh := h.Register("token-a")
	_, viewerBCh := h.Register("token-b")

	evt := ports.Event{
		EventID:    "01960000-0000-7000-8000-000000000001",
		EventType:  "board.state_changed",
		Version:    1,
		OccurredAt: time.Now().UTC(),
	}

	h.Broadcast(evt)

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

// TestHub_CloseToken_ClosesExactlyThatTokensConnections covers T4's second
// DoD line and is what makes T9/T12's AC-11 revoke-closes-SSE test possible:
// CloseToken(token) must close every connection registered under that token
// and must NOT touch connections under a different token or the team-editor
// connection.
func TestHub_CloseToken_ClosesExactlyThatTokensConnections(t *testing.T) {
	h := infra.NewHub()

	_, tokenAConn1 := h.Register("token-a")
	_, tokenAConn2 := h.Register("token-a")
	_, tokenBConn := h.Register("token-b")
	_, teamConn := h.Register("")

	h.CloseToken("token-a")

	requireClosed(t, tokenAConn1, "token-a connection #1")
	requireClosed(t, tokenAConn2, "token-a connection #2")
	requireOpenAndIdle(t, tokenBConn, "token-b connection")
	requireOpenAndIdle(t, teamConn, "team-editor connection")
}

// TestHub_ConcurrentRegisterBroadcastUnregister_NoRace exercises the
// concurrent-safety DoD line: register/unregister/broadcast from multiple
// goroutines simultaneously must not race (run with `go test -race`).
func TestHub_ConcurrentRegisterBroadcastUnregister_NoRace(t *testing.T) {
	h := infra.NewHub()

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(n int) {
			defer wg.Done()
			id, ch := h.Register("token-race")
			h.Broadcast(ports.Event{
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
			h.Unregister("token-race", id)
		}(i)
	}

	wg.Wait()
	h.CloseToken("token-race")
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
		require.Truef(t, ok, "%s: expected the channel to remain open after CloseToken(\"token-a\"), but it was closed", label)
		t.Fatalf("%s: unexpectedly received a value on an idle connection", label)
	default:
		// No value pending and not closed — correct: this connection was
		// never touched by CloseToken("token-a").
	}
}
