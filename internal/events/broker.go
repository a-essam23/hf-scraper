// Path: internal/events/broker.go
package events

import "sync"

// Event represents a message passed through the broker.
type Event struct {
	Topic string
	Data  any
}

// Broker implements a simple in-memory pub/sub system.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

// NewBroker creates a new event broker.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe creates a new subscription to a topic.
// It returns a read-only channel where events for that topic will be sent.
func (b *Broker) Subscribe(topic string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 1) // Buffered channel to prevent blocking publishers
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	return ch
}

// Publish sends an event to all subscribers of a topic.
func (b *Broker) Publish(topic string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{Topic: topic, Data: data}
	if subscribers, found := b.subscribers[topic]; found {
		for _, ch := range subscribers {
			// Non-blocking send
			select {
			case ch <- event:
			default:
				// Subscriber is not ready, drop the event to avoid blocking.
			}
		}
	}
}