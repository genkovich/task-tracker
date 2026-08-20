// Package ports defines the interfaces app depends on and the DTOs/handlers
// exposed over HTTP for the board module.
package ports

import "time"

// Event is the board.state_changed.v1 event shape (contracts/events.md):
// a signal-only payload — no data field — telling a live client to re-fetch
// board state over REST.
type Event struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Version    int       `json:"version"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Broadcaster is what app (T5/T6) calls after a mutation to notify every
// live SSE connection, and what T9's revoke handler calls to close exactly
// one public-link token's connections (sad.md §6, events.md "Connection
// lifecycle").
type Broadcaster interface {
	// Broadcast delivers evt to every registered connection: the
	// team-editor connections and every public-viewer connection,
	// regardless of token.
	Broadcast(evt Event)

	// CloseToken closes every connection registered under token, leaving
	// team-editor connections and connections under other tokens open.
	CloseToken(token string)
}
