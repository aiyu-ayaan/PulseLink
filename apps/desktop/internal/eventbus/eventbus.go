// Package eventbus is a tiny in-process publish/subscribe bus.
//
// Services never call each other directly; they publish events and subscribe to
// the topics they care about. That keeps modules decoupled per the project's
// architecture rules.
package eventbus

import "sync"

// Event is a topic plus an arbitrary payload.
type Event struct {
	Topic   string
	Payload any
}

// Handler receives published events for a topic it subscribed to.
type Handler func(Event)

// Bus is a concurrency-safe fan-out bus.
type Bus struct {
	mu     sync.RWMutex
	nextID int
	subs   map[string]map[int]Handler
}

// New creates an empty Bus.
func New() *Bus {
	return &Bus{subs: make(map[string]map[int]Handler)}
}

// Subscribe registers h for topic and returns a function that unsubscribes it.
func (b *Bus) Subscribe(topic string, h Handler) (unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[int]Handler)
	}
	b.subs[topic][id] = h
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if m := b.subs[topic]; m != nil {
			delete(m, id)
		}
	}
}

// Publish delivers e to every handler subscribed to e.Topic.
//
// ponytail: delivery is synchronous in the caller's goroutine. A handler that
// blocks blocks the publisher. Move to a worker queue if a handler ever needs
// to do slow work; today they all just forward to clients.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, 0, len(b.subs[e.Topic]))
	for _, h := range b.subs[e.Topic] {
		handlers = append(handlers, h)
	}
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}
