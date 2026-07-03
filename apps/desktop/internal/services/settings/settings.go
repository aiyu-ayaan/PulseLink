package settings

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log     *slog.Logger
	bus     *eventbus.Bus
	cfgPath string
}

func New(log *slog.Logger, bus *eventbus.Bus, cfgPath string) *Service {
	return &Service{
		log:     log,
		bus:     bus,
		cfgPath: cfgPath,
	}
}

func (s *Service) Name() string {
	return "settings"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("settings service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("settings service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		cfg, err := config.Load(s.cfgPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	case "set":
		var newCfg config.Config
		if err := req.Bind(&newCfg); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed settings payload"}
		}
		
		err := config.Save(s.cfgPath, newCfg)
		if err != nil {
			return nil, err
		}
		s.log.Info("settings saved successfully")
		return newCfg, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown settings action"}
	}
}
