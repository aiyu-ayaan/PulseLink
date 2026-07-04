package devices

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

type Service struct {
	log *slog.Logger
	bus *eventbus.Bus
	hub *transport.Hub
}

func New(log *slog.Logger, bus *eventbus.Bus, hub *transport.Hub) *Service {
	return &Service{
		log: log,
		bus: bus,
		hub: hub,
	}
}

func (s *Service) Name() string {
	return "devices"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("devices service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("devices service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		return s.hub.ConnectedDevicesInfo(), nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown devices action"}
	}
}
