package notification

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
	return "notification"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("notification service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("notification service stopping")
	return nil
}

type ToastPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "toast":
		var payload ToastPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed toast payload"}
		}
		return nil, s.ShowToast(payload.Title, payload.Message)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown notification action"}
	}
}
