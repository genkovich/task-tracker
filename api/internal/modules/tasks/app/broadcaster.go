package app

import "sync"

// BoardEvent is a change on the board — a card mutation or a public-link
// state change — fanned out to every subscriber (ADR-0001).
type BoardEvent struct {
	Type    string
	Payload any
}

const (
	EventCardCreated   = "card.created"
	EventCardUpdated   = "card.updated"
	EventCardMoved     = "card.moved"
	EventCardDeleted   = "card.deleted"
	EventLinkGenerated = "link.generated"
	EventLinkDisabled  = "link.disabled"
)

const subscriberBufferSize = 16

// Broadcaster is a small in-process pub/sub: every mutation publishes one
// event, and every open SSE connection subscribes to receive it. It works
// only within a single api process (sad.md §7/§11 accepted debt).
type Broadcaster struct {
	mu   sync.Mutex
	subs map[int]chan BoardEvent
	next int
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subs: make(map[int]chan BoardEvent)}
}

// Subscribe registers a new listener and returns its channel plus an
// unsubscribe func the caller must call when done (e.g. on SSE disconnect).
func (b *Broadcaster) Subscribe() (<-chan BoardEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.next
	b.next++
	ch := make(chan BoardEvent, subscriberBufferSize)
	b.subs[id] = ch

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if existing, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(existing)
		}
	}
	return ch, unsubscribe
}

// Publish fans the event out to every current subscriber. A subscriber whose
// buffer is full is skipped (non-blocking send) rather than stalling every
// other subscriber — best-effort delivery, not a durable queue.
func (b *Broadcaster) Publish(event BoardEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// SubscriberCount reports how many listeners are currently attached — used
// for the open-connection observability gauge (sad.md §7).
func (b *Broadcaster) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}
