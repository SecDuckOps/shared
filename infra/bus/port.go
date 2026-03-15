package bus

import "context"

// Envelope for events transported through the bus.
type Message struct {
	Topic   string
	Payload []byte
}

// Contract for publishing events.
type EventPublisher interface {

	// Send an event to a topic.
	Publish(ctx context.Context, msg Message) error
}

// Contract for subscribing to events.
type EventSubscriber interface {

	// Open a subscription and receive messages through a read-only channel.
	// Returns a cleanup function to release resources when finished.
	Subscribe(ctx context.Context, topic string) (<-chan Message, func(), error)
}

// Combined publish/subscribe dependency.
type EventBus interface {
	EventPublisher
	EventSubscriber
}
