package agent

import (
	"sync"
)

// EventBus is a simple publish-subscribe event bus for agent events
type EventBus struct {
	mu        sync.RWMutex
	listeners []chan AgentEvent
	closed    bool
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make([]chan AgentEvent, 0),
	}
}

// Subscribe creates a new subscription to the event bus
// Returns a channel that will receive events
func (bus *EventBus) Subscribe(bufferSize int) <-chan AgentEvent {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.closed {
		// Return a closed channel if the bus is already closed
		ch := make(chan AgentEvent)
		close(ch)
		return ch
	}

	ch := make(chan AgentEvent, bufferSize)
	bus.listeners = append(bus.listeners, ch)
	return ch
}

// Publish publishes an event to all subscribers
func (bus *EventBus) Publish(event AgentEvent) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	if bus.closed {
		return
	}

	// Send to all listeners
	for _, listener := range bus.listeners {
		select {
		case listener <- event:
		default:
			// Skip if channel is full (non-blocking send)
		}
	}
}

// Close closes the event bus and all listener channels
func (bus *EventBus) Close() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.closed {
		return
	}

	bus.closed = true

	// Close all listener channels
	for _, listener := range bus.listeners {
		close(listener)
	}

	bus.listeners = nil
}

// IsClosed returns whether the event bus is closed
func (bus *EventBus) IsClosed() bool {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	return bus.closed
}

// ListenerCount returns the number of active listeners
func (bus *EventBus) ListenerCount() int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	return len(bus.listeners)
}
