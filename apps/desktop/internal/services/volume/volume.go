package volume

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
	return "volume"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("volume service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("volume service stopping")
	return nil
}

type VolumeState struct {
	Level int  `json:"level"` // 0-100
	Muted bool `json:"muted"`
}

type SetVolumePayload struct {
	Level int `json:"level"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "up":
		return nil, s.VolumeUp()
	case "down":
		return nil, s.VolumeDown()
	case "mute":
		return nil, s.VolumeMute()
	case "get":
		return s.GetVolume()
	case "set":
		var payload SetVolumePayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed set volume payload"}
		}
		return nil, s.SetVolume(payload.Level)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown volume action"}
	}
}
