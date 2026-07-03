package filetransfer

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"

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
	return "filetransfer"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("filetransfer service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("filetransfer service stopping")
	return nil
}

type UploadPayload struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"` // base64 encoded
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "upload":
		var payload UploadPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed upload payload"}
		}
		
		data, err := base64.StdEncoding.DecodeString(payload.Content)
		if err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "invalid base64 content"}
		}

		userProfile := os.Getenv("USERPROFILE")
		var downloadsDir string
		if userProfile != "" {
			downloadsDir = filepath.Join(userProfile, "Downloads", "PulseLink")
		} else {
			downloadsDir = filepath.Join(".", "downloads")
		}

		if err := os.MkdirAll(downloadsDir, 0755); err != nil {
			return nil, err
		}

		targetPath := filepath.Join(downloadsDir, filepath.Base(payload.FileName))
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return nil, err
		}

		s.log.Info("file saved successfully", "path", targetPath)
		return map[string]string{"path": targetPath}, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown filetransfer action"}
	}
}
