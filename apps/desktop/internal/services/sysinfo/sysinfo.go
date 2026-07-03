package sysinfo

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
	return "sysinfo"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("sysinfo service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("sysinfo service stopping")
	return nil
}

type SysInfoState struct {
	Hostname     string  `json:"hostname"`
	OS           string  `json:"os"`
	CPUUsage     float64 `json:"cpuUsage"` // percentage
	RAMTotal     uint64  `json:"ramTotal"` // MB
	RAMFree      uint64  `json:"ramFree"`  // MB
	BatteryPct   int     `json:"batteryPct"`
	IsCharging   bool    `json:"isCharging"`
	MonitorCount int     `json:"monitorCount"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		return s.GetSysInfo()
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown sysinfo action"}
	}
}
