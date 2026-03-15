// Package bustest provides contract-test helpers for EventBus implementations.
// Any adapter (in-memory, NATS, Redis, …) can call RunEventBusContract to
// prove it satisfies the EventBus interface.
package bustest

import (
	"context"
	"testing"
	"time"

	"github.com/SecDuckOps/shared/infra/bus"
)

// RunEventBusContract exercises the core behaviours of an EventBus:
// publish/subscribe round-trip, cleanup semantics, and multi-subscriber delivery.
func RunEventBusContract(t *testing.T, b bus.EventBus) {
	t.Helper()
	ctx := context.Background()

	t.Run("Publish and Subscribe round-trip", func(t *testing.T) {
		topic := "test.roundtrip"

		ch, cleanup, err := b.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
		defer cleanup()

		msg := bus.Message{Topic: topic, Payload: []byte("hello")}
		if err := b.Publish(ctx, msg); err != nil {
			t.Fatalf("Publish: %v", err)
		}

		select {
		case got := <-ch:
			if got.Topic != topic {
				t.Fatalf("expected topic %q, got %q", topic, got.Topic)
			}
			if string(got.Payload) != "hello" {
				t.Fatalf("expected payload %q, got %q", "hello", string(got.Payload))
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for message")
		}
	})

	t.Run("Cleanup stops delivery", func(t *testing.T) {
		topic := "test.cleanup"

		ch, cleanup, err := b.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("Subscribe: %v", err)
		}

		// Explicitly call cleanup before publishing.
		cleanup()

		_ = b.Publish(ctx, bus.Message{Topic: topic, Payload: []byte("after-cleanup")})

		select {
		case _, ok := <-ch:
			if ok {
				t.Fatal("expected channel to be closed or no message after cleanup")
			}
		case <-time.After(200 * time.Millisecond):
			// No message received — correct behaviour.
		}
	})

	t.Run("Multiple subscribers on same topic", func(t *testing.T) {
		topic := "test.multi"

		ch1, cleanup1, err := b.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("Subscribe 1: %v", err)
		}
		defer cleanup1()

		ch2, cleanup2, err := b.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("Subscribe 2: %v", err)
		}
		defer cleanup2()

		msg := bus.Message{Topic: topic, Payload: []byte("fan-out")}
		if err := b.Publish(ctx, msg); err != nil {
			t.Fatalf("Publish: %v", err)
		}

		for i, ch := range []<-chan bus.Message{ch1, ch2} {
			select {
			case got := <-ch:
				if string(got.Payload) != "fan-out" {
					t.Fatalf("subscriber %d: expected payload %q, got %q", i+1, "fan-out", string(got.Payload))
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("subscriber %d: timed out waiting for message", i+1)
			}
		}
	})
}
