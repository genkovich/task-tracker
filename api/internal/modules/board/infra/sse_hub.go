// Package infra provides concrete adapters for the board module (Postgres
// repo, in-process SSE hub).
package infra

import (
	"sync"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/modules/board/ports"
)

// eventBuffer is the per-connection channel buffer. Delivery is best-effort,
// at-most-once (contracts/events.md "Delivery") — a slow consumer that
// hasn't drained its buffer simply misses the broadcast, it never blocks the
// broadcaster.
const eventBuffer = 1

// connID identifies one registered connection within its board+token bucket,
// so Unregister can remove exactly one connection among several under the
// same bucket.
type connID uint64

// Hub is an in-process, per-API-instance registry of SSE connections
// (ADR-0002: no message bus), board-scoped (boards BRD-05): connections are
// bucketed by board, then by token within the board — team-editor
// connections under the empty-string token, public-viewer connections under
// their public-link token — so Broadcast reaches exactly one board's
// connections and CloseToken can synchronously close exactly one token's
// connections on revoke (sad.md §6, events.md "Connection lifecycle").
type Hub struct {
	mu      sync.Mutex
	nextID  connID
	byBoard map[uuid.UUID]map[string]map[connID]chan ports.Event
}

// NewHub constructs an empty Hub.
func NewHub() *Hub {
	return &Hub{
		byBoard: make(map[uuid.UUID]map[string]map[connID]chan ports.Event),
	}
}

// Register adds a new connection under boardID's token bucket (empty string
// for a team-editor connection) and returns its id — for a later Unregister
// — and the receive-only channel that future Broadcast calls deliver events
// to.
func (h *Hub) Register(boardID uuid.UUID, token string) (connID, <-chan ports.Event) { //nolint:revive // Subscribe below is the type-safe public API; Register/Unregister are the package-internal pair it composes
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID

	buckets, ok := h.byBoard[boardID]
	if !ok {
		buckets = make(map[string]map[connID]chan ports.Event)
		h.byBoard[boardID] = buckets
	}

	conns, ok := buckets[token]
	if !ok {
		conns = make(map[connID]chan ports.Event)
		buckets[token] = conns
	}

	ch := make(chan ports.Event, eventBuffer)
	conns[id] = ch

	return id, ch
}

// Unregister removes and closes the connection id registered under boardID's
// token bucket. It is a no-op if the connection is already gone (e.g.
// CloseToken already removed it).
func (h *Hub) Unregister(boardID uuid.UUID, token string, id connID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	buckets, ok := h.byBoard[boardID]
	if !ok {
		return
	}

	conns, ok := buckets[token]
	if !ok {
		return
	}

	ch, ok := conns[id]
	if !ok {
		return
	}

	delete(conns, id)
	if len(conns) == 0 {
		delete(buckets, token)
	}
	if len(buckets) == 0 {
		delete(h.byBoard, boardID)
	}
	close(ch)
}

// Broadcast delivers evt to every connection registered under boardID — its
// team-editor connections and its public-viewer connections alike; other
// boards' connections never see it (boards BRD-05). Delivery is best-effort:
// a connection whose buffer is already full does not receive this event and
// does not block the broadcaster (events.md "Delivery: best-effort,
// at-most-once").
func (h *Hub) Broadcast(boardID uuid.UUID, evt ports.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, conns := range h.byBoard[boardID] {
		for _, ch := range conns {
			select {
			case ch <- evt:
			default:
			}
		}
	}
}

// Subscribe registers a new connection under boardID's token bucket and
// returns its event channel plus a func that unregisters it — the
// port-facing counterpart to Register/Unregister (ports.SSERegistry) for
// consumers outside this package, which cannot name the unexported connID
// type Register/Unregister use directly.
func (h *Hub) Subscribe(boardID uuid.UUID, token string) (<-chan ports.Event, func()) {
	id, ch := h.Register(boardID, token)
	return ch, func() { h.Unregister(boardID, token, id) }
}

// CloseToken closes exactly the connections registered under boardID's token
// bucket, leaving the board's team-editor connections and every other
// board's connections untouched (events.md "Connection lifecycle" — revoke
// must close already-open SSE connections synchronously, not just block new
// ones).
func (h *Hub) CloseToken(boardID uuid.UUID, token string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	buckets, ok := h.byBoard[boardID]
	if !ok {
		return
	}

	conns, ok := buckets[token]
	if !ok {
		return
	}

	for _, ch := range conns {
		close(ch)
	}
	delete(buckets, token)
	if len(buckets) == 0 {
		delete(h.byBoard, boardID)
	}
}
