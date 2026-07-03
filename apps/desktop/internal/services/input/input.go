package input

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log *slog.Logger
	bus *eventbus.Bus
}

func New(log *slog.Logger, bus *eventbus.Bus) *Service {
	return &Service{
		log: log,
		bus: bus,
	}
}

func (s *Service) Name() string {
	return "input"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("input service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("input service stopping")
	return nil
}

type MouseMovePayload struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Relative bool `json:"relative"`
}

type MouseClickPayload struct {
	Button string `json:"button"` // "left", "right", "middle"
}

type KeyPressPayload struct {
	Key string `json:"key"` // Key name or character
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "mouse_move":
		var payload MouseMovePayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed mouse_move payload"}
		}
		return nil, s.MouseMove(payload.X, payload.Y, payload.Relative)
	case "mouse_click":
		var payload MouseClickPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed mouse_click payload"}
		}
		return nil, s.MouseClick(payload.Button)
	case "keypress":
		var payload KeyPressPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed keypress payload"}
		}
		return nil, s.KeyPress(payload.Key)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown input action"}
	}
}
