package pairing

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
	store *storage.Store
	hub   *transport.Hub
}

func New(log *slog.Logger, bus *eventbus.Bus, store *storage.Store, hub *transport.Hub) *Service {
	return &Service{
		log:   log,
		bus:   bus,
		store: store,
		hub:   hub,
	}
}

func (s *Service) Name() string {
	return "pairing"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("pairing service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("pairing service stopping")
	return nil
}

type DeviceActionPayload struct {
	DeviceID string `json:"deviceId"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "list":
		all, err := s.store.Devices.List()
		if err != nil {
			return nil, err
		}
		var pending []transport.DeviceInfo
		for _, d := range all {
			if !d.Trusted {
				pending = append(pending, transport.DeviceInfo{
					ID:           d.ID,
					Name:         d.Name,
					Capabilities: d.Capabilities,
				})
			}
		}
		return pending, nil

	case "pending":
		all, err := s.store.Devices.List()
		if err != nil {
			return nil, err
		}
		count := 0
		var pending []transport.DeviceInfo
		for _, d := range all {
			if !d.Trusted {
				count++
				pending = append(pending, transport.DeviceInfo{
					ID:           d.ID,
					Name:         d.Name,
					Capabilities: d.Capabilities,
				})
			}
		}
		return map[string]any{
			"count":   count,
			"devices": pending,
		}, nil

	case "accept":
		var payload DeviceActionPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed pairing payload"}
		}

		s.log.Info("pairing accepted", "device", payload.DeviceID)
		if err := s.store.Devices.SetTrusted(payload.DeviceID, true); err != nil {
			return nil, err
		}

		s.bus.Publish(eventbus.Event{
			Topic:   "pairing.approved",
			Payload: payload.DeviceID,
		})
		return map[string]string{"status": "approved", "deviceId": payload.DeviceID}, nil

	case "reject":
		var payload DeviceActionPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed pairing payload"}
		}

		s.log.Info("pairing rejected", "device", payload.DeviceID)
		_ = s.store.Devices.Delete(payload.DeviceID)

		s.bus.Publish(eventbus.Event{
			Topic:   "pairing.rejected",
			Payload: payload.DeviceID,
		})
		return map[string]string{"status": "rejected", "deviceId": payload.DeviceID}, nil

	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown pairing action"}
	}
}
