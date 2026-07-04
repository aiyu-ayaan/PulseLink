package media

import (
	"context"
	"log/slog"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type MediaState struct {
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	AlbumTitle string `json:"albumTitle"`
	Status     string `json:"status"` // e.g. "Playing", "Paused", "Stopped"
}

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
	return "media"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("media service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("media service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	var err error
	switch req.Action {
	case "play_pause":
		err = s.PlayPause()
	case "next":
		err = s.Next()
	case "previous":
		err = s.Previous()
	case "stop":
		err = s.StopMedia()
	case "get":
		return s.GetMediaState()
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown media action"}
	}

	if err != nil {
		return nil, err
	}

	// Fetch and broadcast the updated media state asynchronously
	go func() {
		time.Sleep(500 * time.Millisecond)
		state, err := s.GetMediaState()
		if err == nil {
			s.bus.Publish(eventbus.Event{
				Topic:   "media.changed",
				Payload: state,
			})
		}
	}()

	return nil, nil
}

