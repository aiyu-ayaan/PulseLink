// Package transport is the WebSocket networking layer: it accepts client
// connections, performs the capability handshake, keeps connections alive with
// heartbeats, and routes protocol messages to registered handlers.
//
// It depends only on the protocol package and small interfaces (Authenticator,
// Handler) so it stays decoupled from storage and the feature services.
package transport

import (
	"context"
	"sync"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

// Handler processes requests for a single capability. Feature services
// implement it and register with the Router under their capability name.
type Handler interface {
	// Handle processes a request and returns a response payload (any) or an
	// error. Returning a *protocol.Error controls the response code; any other
	// error becomes CodeInternal.
	Handle(ctx context.Context, req protocol.Envelope) (any, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, req protocol.Envelope) (any, error)

// Handle calls f.
func (f HandlerFunc) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	return f(ctx, req)
}

// Router dispatches requests to handlers by capability.
type Router struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRouter creates an empty router.
func NewRouter() *Router {
	return &Router{handlers: make(map[string]Handler)}
}

// Register binds a handler to a capability, replacing any previous one.
func (r *Router) Register(capability string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[capability] = h
}

// Capabilities returns the registered capability names.
func (r *Router) Capabilities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.handlers))
	for c := range r.handlers {
		out = append(out, c)
	}
	return out
}

// Dispatch routes req to its handler and returns a response envelope. It never
// returns an error: failures are encoded as an error response so the caller can
// always write something back.
func (r *Router) Dispatch(ctx context.Context, req protocol.Envelope) protocol.Envelope {
	r.mu.RLock()
	h, ok := r.handlers[req.Capability]
	r.mu.RUnlock()
	if !ok {
		return protocol.NewErrorResponse(req, protocol.CodeUnsupported, "unknown capability: "+req.Capability)
	}

	result, err := h.Handle(ctx, req)
	if err != nil {
		if pe, ok := err.(*protocol.Error); ok {
			return protocol.NewErrorResponse(req, pe.Code, pe.Message)
		}
		return protocol.NewErrorResponse(req, protocol.CodeInternal, err.Error())
	}
	resp, err := protocol.NewResponse(req, result)
	if err != nil {
		return protocol.NewErrorResponse(req, protocol.CodeInternal, err.Error())
	}
	return resp
}
