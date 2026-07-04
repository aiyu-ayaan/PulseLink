package volume

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log    *slog.Logger
	bus    *eventbus.Bus
	cancel context.CancelFunc
	mu     sync.Mutex
	last   VolumeState

	// mock holds volume state for non-Windows dev builds (see volume_other.go).
	// Unused on Windows, where real Core Audio state is read live.
	mock VolumeState
}

func New(log *slog.Logger, bus *eventbus.Bus) *Service {
	return &Service{
		log:  log,
		bus:  bus,
		mock: VolumeState{Level: 75, Muted: false},
	}
}

func (s *Service) Name() string {
	return "volume"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("volume service starting")
	
	// Get initial state
	if st, err := s.GetVolume(); err == nil {
		s.last = st
	}

	monCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.monitorLoop(monCtx)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("volume service stopping")
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *Service) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current, err := s.GetVolume()
			if err != nil {
				continue
			}

			s.mu.Lock()
			if current.Level != s.last.Level || current.Muted != s.last.Muted {
				s.last = current
				s.mu.Unlock()
				s.log.Debug("volume change detected", "level", current.Level, "muted", current.Muted)
				s.bus.Publish(eventbus.Event{
					Topic:   "volume.changed",
					Payload: current,
				})
			} else {
				s.mu.Unlock()
			}
		}
	}
}

type VolumeState struct {
	Level int  `json:"level"` // 0-100
	Muted bool `json:"muted"`
}

type SetVolumePayload struct {
	Level int `json:"level"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	var st VolumeState
	var err error

	switch req.Action {
	case "up":
		st, err = s.VolumeUp()
	case "down":
		st, err = s.VolumeDown()
	case "mute":
		st, err = s.VolumeMute()
	case "get":
		return s.GetVolume()
	case "set":
		var payload SetVolumePayload
		if err = req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed set volume payload"}
		}
		st, err = s.SetVolume(payload.Level)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown volume action"}
	}

	if err == nil {
		s.mu.Lock()
		s.last = st
		s.mu.Unlock()
		s.bus.Publish(eventbus.Event{
			Topic:   "volume.changed",
			Payload: st,
		})
	}
	return st, err
}
