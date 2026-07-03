package apps

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
	return "apps"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("apps service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("apps service stopping")
	return nil
}

type AppItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type LaunchPayload struct {
	Name string `json:"name"`
}

var predefinedApps = []AppItem{
	{Name: "Notepad", Path: "notepad.exe"},
	{Name: "Calculator", Path: "calc.exe"},
	{Name: "Task Manager", Path: "taskmgr.exe"},
	{Name: "Command Prompt", Path: "cmd.exe"},
	{Name: "Paint", Path: "mspaint.exe"},
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "list":
		return predefinedApps, nil
	case "launch":
		var payload LaunchPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed launch app payload"}
		}
		return nil, s.LaunchApp(payload.Name)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown apps action"}
	}
}
