package eventbus

import (
	"sync"
	"testing"
)

func TestPublishSubscribe(t *testing.T) {
	b := New()
	var mu sync.Mutex
	var got []any

	unsub := b.Subscribe("media.play", func(e Event) {
		mu.Lock()
		got = append(got, e.Payload)
		mu.Unlock()
	})

	b.Publish(Event{Topic: "media.play", Payload: 1})
	b.Publish(Event{Topic: "other", Payload: 2}) // no subscriber, ignored
	unsub()
	b.Publish(Event{Topic: "media.play", Payload: 3}) // after unsub, dropped

	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("want [1], got %v", got)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := New()
	n := 0
	b.Subscribe("t", func(Event) { n++ })
	b.Subscribe("t", func(Event) { n++ })
	b.Publish(Event{Topic: "t"})
	if n != 2 {
		t.Fatalf("want 2 handlers called, got %d", n)
	}
}
