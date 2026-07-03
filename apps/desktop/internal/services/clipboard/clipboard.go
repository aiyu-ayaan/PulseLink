package clipboard

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log      *slog.Logger
	bus      *eventbus.Bus
	cancel   context.CancelFunc
	mu       sync.Mutex
	lastText string
}

func New(log *slog.Logger, bus *eventbus.Bus) *Service {
	return &Service{
		log: log,
		bus: bus,
	}
}

func (s *Service) Name() string {
	return "clipboard"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("clipboard service starting")
	
	monCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.monitorLoop(monCtx)
	
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("clipboard service stopping")
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

type ClipboardPayload struct {
	Text string `json:"text"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		text, err := s.GetText()
		if err != nil {
			return nil, err
		}
		return ClipboardPayload{Text: text}, nil
	case "set":
		var payload ClipboardPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed clipboard text payload"}
		}
		err := s.SetText(payload.Text)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.lastText = payload.Text // prevent echo loop
		s.mu.Unlock()
		return nil, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown clipboard action"}
	}
}

func (s *Service) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	// Initialize lastText with current clipboard
	if val, err := s.GetText(); err == nil {
		s.mu.Lock()
		s.lastText = val
		s.mu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentText, err := s.GetText()
			if err != nil {
				continue
			}
			s.mu.Lock()
			if currentText != s.lastText && currentText != "" {
				s.lastText = currentText
				s.mu.Unlock()
				s.log.Debug("clipboard change detected", "len", len(currentText))
				
				// Publish clipboard change on the event bus
				s.bus.Publish(eventbus.Event{
					Topic:   "clipboard.changed",
					Payload: ClipboardPayload{Text: currentText},
				})
			} else {
				s.mu.Unlock()
			}
		}
	}
}
