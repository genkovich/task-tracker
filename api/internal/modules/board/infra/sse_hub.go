// Package infra provides concrete adapters for the board module (Postgres
// repo, in-process SSE hub).
package infra

import (
	"sync"

	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// eventBuffer is the per-connection channel buffer. Delivery is best-effort,
// at-most-once (contracts/events.md "Delivery") — a slow consumer that
// hasn't drained its buffer simply misses the broadcast, it never blocks the
// broadcaster.
const eventBuffer = 1

// connID identifies one registered connection within its token's (or the
// team-editor "") registry, so Unregister can remove exactly one connection
// among several under the same token.
type connID uint64

// Hub is an in-process, per-API-instance registry of SSE connections
// (ADR-0002: no message bus). Team-editor connections are registered under
// the empty-string token; public-viewer connections are grouped by their
// public-link token so CloseToken can synchronously close exactly one
// token's connections on revoke (sad.md §6, events.md "Connection
// lifecycle").
type Hub struct {
	mu      sync.Mutex
	nextID  connID
	byToken map[string]map[connID]chan ports.Event
}

// NewHub constructs an empty Hub.
func NewHub() *Hub {
	return &Hub{
		byToken: make(map[string]map[connID]chan ports.Event),
	}
}

// Register adds a new connection under token (empty string for the
// team-editor connection) and returns its id — for a later Unregister — and
// the receive-only channel that future Broadcast calls deliver events to.
func (h *Hub) Register(token string) (connID, <-chan ports.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID

	conns, ok := h.byToken[token]
	if !ok {
		conns = make(map[connID]chan ports.Event)
		h.byToken[token] = conns
	}

	ch := make(chan ports.Event, eventBuffer)
	conns[id] = ch

	return id, ch
}

// Unregister removes and closes the connection id registered under token.
// It is a no-op if the connection is already gone (e.g. CloseToken already
// removed it).
func (h *Hub) Unregister(token string, id connID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.byToken[token]
	if !ok {
		return
	}

	ch, ok := conns[id]
	if !ok {
		return
	}

	delete(conns, id)
	if len(conns) == 0 {
		delete(h.byToken, token)
	}
	close(ch)
}

// Broadcast delivers evt to every registered connection, team-editor and
// every public-viewer token alike. Delivery is best-effort: a connection
// whose buffer is already full does not receive this event and does not
// block the broadcaster (events.md "Delivery: best-effort, at-most-once").
func (h *Hub) Broadcast(evt ports.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, conns := range h.byToken {
		for _, ch := range conns {
			select {
			case ch <- evt:
			default:
			}
		}
	}
}

// Subscribe registers a new connection under token and returns its event
// channel plus a func that unregisters it — the port-facing counterpart to
// Register/Unregister (T9's ports.SSERegistry) for consumers outside this
// package, which cannot name the unexported connID type Register/Unregister
// use directly.
func (h *Hub) Subscribe(token string) (<-chan ports.Event, func()) {
	id, ch := h.Register(token)
	return ch, func() { h.Unregister(token, id) }
}

// CloseToken closes exactly the connections registered under token, leaving
// team-editor connections and every other token's connections untouched
// (events.md "Connection lifecycle" — revoke must close already-open SSE
// connections synchronously, not just block new ones).
func (h *Hub) CloseToken(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.byToken[token]
	if !ok {
		return
	}

	for _, ch := range conns {
		close(ch)
	}
	delete(h.byToken, token)
}
