package devices

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

type Service struct {
	log   *slog.Logger
	bus   *eventbus.Bus
	hub   *transport.Hub
	store *storage.Store
}

func New(log *slog.Logger, bus *eventbus.Bus, hub *transport.Hub, store *storage.Store) *Service {
	return &Service{
		log:   log,
		bus:   bus,
		hub:   hub,
		store: store,
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

type DeviceHistoryItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Trusted      bool     `json:"trusted"`
	Online       bool     `json:"online"`
	PairedAt     int64    `json:"pairedAt"`
	LastSeen     int64    `json:"lastSeen"`
	Capabilities []string `json:"capabilities"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		return s.hub.ConnectedDevicesInfo(), nil
	case "history":
		all, err := s.store.Devices.List()
		if err != nil {
			return nil, err
		}

		connectedMap := make(map[string]bool)
		for _, info := range s.hub.ConnectedDevicesInfo() {
			connectedMap[info.ID] = true
		}

		history := make([]DeviceHistoryItem, 0, len(all))
		for _, d := range all {
			history = append(history, DeviceHistoryItem{
				ID:           d.ID,
				Name:         d.Name,
				Trusted:      d.Trusted,
				Online:       connectedMap[d.ID],
				PairedAt:     d.PairedAt.Unix(),
				LastSeen:     d.LastSeen.Unix(),
				Capabilities: d.Capabilities,
			})
		}
		return history, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown devices action"}
	}
}
