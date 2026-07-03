package transport

import (
	"sync"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

// Hub tracks connected clients and fans messages out to them. It is safe for
// concurrent use.
type Hub struct {
	mu       sync.RWMutex
	byClient map[string]*Client // keyed by connection ID
	byDevice map[string]*Client // keyed by device ID (latest wins)

	// OnChange, if set, is called after a client connects or disconnects with
	// the current connected device IDs. The app uses it to publish presence.
	OnChange func(deviceIDs []string)
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{
		byClient: make(map[string]*Client),
		byDevice: make(map[string]*Client),
	}
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	h.byClient[c.ID] = c
	if c.DeviceID != "" {
		// Drop a stale connection from the same device.
		if old, ok := h.byDevice[c.DeviceID]; ok && old != c {
			old.Close()
			delete(h.byClient, old.ID)
		}
		h.byDevice[c.DeviceID] = c
	}
	h.mu.Unlock()
	h.notify()
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	delete(h.byClient, c.ID)
	if cur, ok := h.byDevice[c.DeviceID]; ok && cur == c {
		delete(h.byDevice, c.DeviceID)
	}
	h.mu.Unlock()
	h.notify()
}

func (h *Hub) notify() {
	if h.OnChange == nil {
		return
	}
	h.OnChange(h.ConnectedDevices())
}

// ConnectedDevices returns the device IDs currently connected.
func (h *Hub) ConnectedDevices() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.byDevice))
	for id := range h.byDevice {
		out = append(out, id)
	}
	return out
}

// Count returns the number of live connections.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.byClient)
}

// SendToDevice queues env for the given device. Returns false if not connected.
func (h *Hub) SendToDevice(deviceID string, env protocol.Envelope) bool {
	h.mu.RLock()
	c := h.byDevice[deviceID]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	return c.Send(env)
}

// Broadcast queues env for every connected client.
func (h *Hub) Broadcast(env protocol.Envelope) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.byClient))
	for _, c := range h.byClient {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.Send(env)
	}
}
