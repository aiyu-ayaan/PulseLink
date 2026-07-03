package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

// ServerInfo describes this backend to connecting clients.
type ServerInfo struct {
	Name    string
	Version string
}

// Config configures the WebSocket server.
type Config struct {
	Addr string      // host:port to listen on
	TLS  *tls.Config // nil for plain ws:// (dev only)
	Info ServerInfo
}

// Server accepts WebSocket connections and wires them to the router and hub.
type Server struct {
	cfg    Config
	log    *slog.Logger
	hub    *Hub
	router *Router
	auth   Authenticator

	http *http.Server
	ln   net.Listener
}

// NewServer constructs a server. hub, router and auth must be non-nil.
func NewServer(cfg Config, log *slog.Logger, hub *Hub, router *Router, auth Authenticator) *Server {
	return &Server{cfg: cfg, log: log, hub: hub, router: router, auth: auth}
}

// handshakeTimeout bounds how long a client has to send its hello.
const handshakeTimeout = 10 * time.Second

// Start begins listening. It returns once the listener is open; serving runs in
// a background goroutine until ctx is cancelled or Stop is called.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Serve static UI assets if compiled
	distDir := "./apps/desktop/frontend/dist"
	if _, err := os.Stat(distDir); err == nil {
		s.log.Info("serving static assets from", "dir", distDir)
		mux.Handle("/", http.FileServer(http.Dir(distDir)))
	} else {
		s.log.Debug("static frontend assets not found, running api mode only")
	}

	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	if s.cfg.TLS != nil {
		ln = tls.NewListener(ln, s.cfg.TLS)
	}
	s.ln = ln
	s.http = &http.Server{Handler: mux}

	go func() {
		s.log.Info("websocket server listening", "addr", s.cfg.Addr, "tls", s.cfg.TLS != nil)
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http serve", "err", err)
		}
	}()
	return nil
}

// Stop gracefully shuts the server down.
func (s *Server) Stop(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// ponytail: no origin check — this is a LAN service reached by native
		// clients, not browsers. Add OriginPatterns if a web client appears.
		InsecureSkipVerify: true,
	})
	if err != nil {
		s.log.Debug("ws accept failed", "err", err)
		return
	}

	client := newClient(uuid.NewString(), conn, s.log)

	if !s.handshake(r.Context(), client) {
		client.Close()
		return
	}

	s.hub.add(client)
	s.log.Info("device connected", "device", client.DeviceID, "name", client.DeviceName,
		"capabilities", client.Capabilities)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.writePump(ctx)
	client.readPump(ctx, s.router) // blocks until disconnect

	s.hub.remove(client)
	s.log.Info("device disconnected", "device", client.DeviceID)
}

// handshake reads the ClientHello, authenticates, negotiates capabilities and
// replies with a ServerWelcome. It returns true when the session is accepted.
func (s *Server) handshake(parent context.Context, c *Client) bool {
	ctx, cancel := context.WithTimeout(parent, handshakeTimeout)
	defer cancel()

	typ, data, err := c.conn.Read(ctx)
	if err != nil || typ != websocket.MessageText {
		return false
	}
	req, err := protocol.Decode(data)
	if err != nil || req.Capability != protocol.CapHandshake || req.Action != protocol.ActionHello {
		s.writeWelcome(ctx, c, req, protocol.ServerWelcome{Accepted: false, Reason: "expected handshake hello"})
		return false
	}

	var hello protocol.ClientHello
	if err := req.Bind(&hello); err != nil {
		s.writeWelcome(ctx, c, req, protocol.ServerWelcome{Accepted: false, Reason: "malformed hello"})
		return false
	}

	res, err := s.auth.Authenticate(hello)
	if err != nil {
		s.log.Error("authenticate", "err", err)
		s.writeWelcome(ctx, c, req, protocol.ServerWelcome{Accepted: false, Reason: "internal auth error"})
		return false
	}
	if !res.Accepted {
		s.writeWelcome(ctx, c, req, protocol.ServerWelcome{Accepted: false, Reason: res.Reason})
		return false
	}

	// Offered = server capabilities, optionally narrowed by device permissions,
	// then intersected with what the client asked for.
	offered := s.router.Capabilities()
	if res.AllowedCapabilities != nil {
		offered = protocol.Negotiate(offered, res.AllowedCapabilities)
	}
	negotiated := protocol.Negotiate(offered, hello.Capabilities)

	c.DeviceID = res.DeviceID
	c.DeviceName = res.DeviceName
	c.Capabilities = negotiated

	welcome := protocol.ServerWelcome{
		ProtocolVersion: protocol.Version,
		ServerName:      s.cfg.Info.Name,
		ServerVersion:   s.cfg.Info.Version,
		Capabilities:    negotiated,
		Accepted:        true,
	}
	return s.writeWelcome(ctx, c, req, welcome)
}

func (s *Server) writeWelcome(ctx context.Context, c *Client, req protocol.Envelope, w protocol.ServerWelcome) bool {
	// Preserve request ID when we have one so the client can correlate.
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	resp, err := protocol.NewResponse(protocol.Envelope{
		ID: req.ID, Capability: protocol.CapHandshake, Action: protocol.ActionWelcome,
	}, w)
	if err != nil {
		return false
	}
	data, err := protocol.Encode(resp)
	if err != nil {
		return false
	}
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return false
	}
	return w.Accepted
}
