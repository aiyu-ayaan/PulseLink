package transport

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

const (
	// sendBuffer bounds per-client outbound queue depth. A slow client that
	// fills it is disconnected rather than allowed to back up the server.
	sendBuffer = 32
	// pingInterval is how often the server pings idle clients.
	pingInterval = 20 * time.Second
	// writeTimeout bounds a single frame write.
	writeTimeout = 10 * time.Second
)

// Client is one live WebSocket connection to a paired device.
type Client struct {
	ID           string   // connection id (uuid)
	DeviceID     string   // resolved during handshake
	DeviceName   string
	Capabilities []string // negotiated for this session
	isAllowed    func(capability string) bool

	conn   *websocket.Conn
	send   chan protocol.Envelope
	log    *slog.Logger
	closeOnce sync.Once
	done      chan struct{}
}

func newClient(id string, conn *websocket.Conn, log *slog.Logger) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		send: make(chan protocol.Envelope, sendBuffer),
		log:  log,
		done: make(chan struct{}),
	}
}

// Send queues env for delivery. It returns false and closes the client if the
// outbound buffer is full (slow/stuck consumer).
func (c *Client) Send(env protocol.Envelope) bool {
	select {
	case <-c.done:
		return false
	case c.send <- env:
		return true
	default:
		c.log.Warn("client send buffer full, dropping connection", "device", c.DeviceID)
		c.Close()
		return false
	}
}

// Close shuts the connection down once.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close(websocket.StatusNormalClosure, "")
	})
}

// writePump serialises all writes (coder/websocket allows one writer) and pings
// on an interval so dead connections are detected.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case env := <-c.send:
			data, err := protocol.Encode(env)
			if err != nil {
				c.log.Error("encode outbound", "err", err)
				continue
			}
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err = c.conn.Write(wctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				c.log.Debug("write failed, closing", "device", c.DeviceID, "err", err)
				c.Close()
				return
			}
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pctx)
			cancel()
			if err != nil {
				c.log.Debug("ping failed, closing", "device", c.DeviceID, "err", err)
				c.Close()
				return
			}
		}
	}
}

// readPump reads frames, dispatches requests, and returns when the connection
// closes. Responses/events from the client (if any) are ignored for now.
func (c *Client) readPump(ctx context.Context, router *Router) {
	defer c.Close()
	for {
		typ, data, err := c.conn.Read(ctx)
		if err != nil {
			return // connection closed or context cancelled
		}
		if typ != websocket.MessageText {
			continue
		}
		req, err := protocol.Decode(data)
		if err != nil {
			c.log.Debug("bad frame", "device", c.DeviceID, "err", err)
			continue
		}
		if req.Type != protocol.TypeRequest {
			continue // only requests are routed; ignore client-originated events
		}
		if !c.allowed(req.Capability) {
			c.Send(protocol.NewErrorResponse(req, protocol.CodeUnauthorized,
				"capability not granted: "+req.Capability))
			continue
		}
		// ponytail: dispatch per-request in its own goroutine so a slow handler
		// (e.g. sysinfo/volume shelling out) doesn't block every other command
		// behind it on this connection. Responses still funnel through the
		// thread-safe send channel; clients correlate by ID so order is fine.
		go c.Send(router.Dispatch(ctx, req))
	}
}

func (c *Client) allowed(capability string) bool {
	if capability == protocol.CapHandshake {
		return true
	}
	if c.isAllowed != nil && !c.isAllowed(capability) {
		return false
	}
	for _, cap := range c.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}
