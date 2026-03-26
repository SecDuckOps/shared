package bus

import (
	"context"
	"sync"
)

// ChannelBus is a pure Go in-memory implementation of the EventBus interface.
type ChannelBus struct {
	subs map[string][]chan Message
	mu   sync.RWMutex
}

// NewChannelBus creates a new, ready-to-use ChannelBus.
func NewChannelBus() *ChannelBus {
	return &ChannelBus{
		subs: make(map[string][]chan Message),
	}
}

// Publish broadcasts a message to all active subscribers on the message's topic.
// It uses a non-blocking fan-out approach: if a subscriber's buffer is full,
// the message is silently dropped.
func (b *ChannelBus) Publish(ctx context.Context, msg Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subs[msg.Topic] {
		// Respect publisher's context cancellation before attempting to send
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Non-blocking send
		select {
		case ch <- msg:
		default:
		}
	}
	return nil
}

// Subscribe returns a read-only channel of messages for the given topic.
// It returns a cleanup function that MUST be called to release resources.
func (b *ChannelBus) Subscribe(ctx context.Context, topic string) (<-chan Message, func(), error) {
	// Buffered channel of size 100 per rule
	ch := make(chan Message, 100)

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()

	var once sync.Once

	// A context specifically to stop the auto-cleanup watcher if cleanup is called manually
	stopCtx, stopCancel := context.WithCancel(context.Background())

	cleanup := func() {
		once.Do(func() {
			stopCancel() // Stop the watcher goroutine

			b.mu.Lock()
			defer b.mu.Unlock()

			// Remove the specific channel from the subscription list
			subs := b.subs[topic]
			for i, subCh := range subs {
				if subCh == ch {
					// Safe slice deletion
					b.subs[topic] = append(subs[:i], subs[i+1:]...)
					break
				}
			}

			// We close under the Lock. Because Publish holds RLock across its loop,
			// it is impossible for Publish to ever attempt sending to a closed channel.
			close(ch)
		})
	}

	// Auto-cleanup watcher:
	// Unregisters the subscriber automatically if their given Context cancels.
	go func() {
		select {
		case <-ctx.Done():
			cleanup()
		case <-stopCtx.Done():
			// Manual cleanup was triggered; exit cleanly
		}
	}()

	return ch, cleanup, nil
}
