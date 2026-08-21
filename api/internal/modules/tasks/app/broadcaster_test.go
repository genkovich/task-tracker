package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/genkovich/task-tracker/api/internal/modules/tasks/app"
)

func TestBroadcaster_FanOutToMultipleSubscribers(t *testing.T) {
	b := app.NewBroadcaster()

	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	b.Publish(app.BoardEvent{Type: app.EventCardMoved})

	for _, ch := range []<-chan app.BoardEvent{ch1, ch2} {
		select {
		case ev := <-ch:
			require.Equal(t, app.EventCardMoved, ev.Type)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}
}

func TestBroadcaster_UnsubscribeStopsDelivery(t *testing.T) {
	b := app.NewBroadcaster()
	ch, unsub := b.Subscribe()
	require.Equal(t, 1, b.SubscriberCount())

	unsub()
	require.Equal(t, 0, b.SubscriberCount())

	// The channel is closed on unsubscribe — receiving yields the zero value
	// with ok=false, never blocks.
	_, ok := <-ch
	require.False(t, ok)
}

func TestBroadcaster_SlowSubscriberDoesNotBlockPublish(t *testing.T) {
	b := app.NewBroadcaster()
	_, unsub := b.Subscribe() // never drained
	defer unsub()

	done := make(chan struct{})
	go func() {
		for range 100 {
			b.Publish(app.BoardEvent{Type: app.EventCardMoved})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full/slow subscriber buffer")
	}
}
