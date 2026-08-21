// Package ports defines the interfaces app depends on and the DTOs/handlers
// exposed over HTTP for the board module.
package ports

import (
	"time"

	"github.com/google/uuid"
)

// Event is the board.state_changed.v1 event shape (contracts/events.md):
// a signal-only payload — no data field — telling a live client to re-fetch
// board state over REST.
type Event struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Broadcaster is what app calls after a mutation to notify the mutated
// board's live SSE connections, and what the revoke path calls to close
// exactly one public-link token's connections (sad.md §6, events.md
// "Connection lifecycle"). Board-scoped (boards BRD-05): events from board A
// never reach board B's connections.
type Broadcaster interface {
	// Broadcast delivers evt to every connection registered under boardID:
	// its team-editor connections and its public-viewer connections. Other
	// boards' connections never see it.
	Broadcast(boardID uuid.UUID, evt Event)

	// CloseToken closes every connection registered under boardID's token
	// bucket, leaving the board's team-editor connections and every other
	// board's connections open.
	CloseToken(boardID uuid.UUID, token string)
}
