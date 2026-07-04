package pairing

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAcceptMarksPendingDeviceTrusted(t *testing.T) {
	store := newTestStore(t)
	bus := eventbus.New()
	svc := New(slog.Default(), bus, store, transport.NewHub(), config.Default())

	if err := store.Devices.Upsert(storage.Device{
		ID:           "android-pixel",
		Name:         "Pixel",
		PublicKey:    "desktop-local",
		Trusted:      false,
		PairedAt:     time.Now(),
		LastSeen:     time.Now(),
		Capabilities: []string{"pairing", "volume"},
	}); err != nil {
		t.Fatalf("seed device: %v", err)
	}

	var approved string
	bus.Subscribe("pairing.approved", func(ev eventbus.Event) {
		approved, _ = ev.Payload.(string)
	})

	req, err := protocol.NewRequest("accept-1", "pairing", "accept", DeviceActionPayload{DeviceID: " android-pixel "})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if _, err := svc.Handle(context.Background(), req); err != nil {
		t.Fatalf("accept: %v", err)
	}

	dev, err := store.Devices.Get("android-pixel")
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if !dev.Trusted {
		t.Fatal("device should be trusted after accept")
	}
	if approved != "android-pixel" {
		t.Fatalf("expected approval event for android-pixel, got %q", approved)
	}
}

func TestAcceptRejectsMissingDeviceID(t *testing.T) {
	store := newTestStore(t)
	svc := New(slog.Default(), eventbus.New(), store, transport.NewHub(), config.Default())

	req, err := protocol.NewRequest("accept-1", "pairing", "accept", DeviceActionPayload{})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_, err = svc.Handle(context.Background(), req)
	if pe, ok := err.(*protocol.Error); !ok || pe.Code != protocol.CodeBadRequest {
		t.Fatalf("want bad_request, got %v", err)
	}
}

func TestAcceptRejectsUnknownDevice(t *testing.T) {
	store := newTestStore(t)
	svc := New(slog.Default(), eventbus.New(), store, transport.NewHub(), config.Default())

	req, err := protocol.NewRequest("accept-1", "pairing", "accept", DeviceActionPayload{DeviceID: "missing"})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_, err = svc.Handle(context.Background(), req)
	if pe, ok := err.(*protocol.Error); !ok || pe.Code != protocol.CodeNotFound {
		t.Fatalf("want not_found, got %v", err)
	}
}

func TestInfoCreatesPairingTokenAndURI(t *testing.T) {
	store := newTestStore(t)
	cfg := config.Default()
	cfg.Server.Host = "192.168.1.103"
	cfg.Server.Port = 9843
	cfg.DeviceName = "PulseLink-PC"
	svc := New(slog.Default(), eventbus.New(), store, transport.NewHub(), cfg)

	req, err := protocol.NewRequest("info-1", "pairing", "info", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	out, err := svc.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	info, ok := out.(Info)
	if !ok {
		t.Fatalf("unexpected info payload: %T", out)
	}
	if info.Host != "192.168.1.103" || info.Port != 9843 || info.Scheme != "ws" {
		t.Fatalf("unexpected connection info: %+v", info)
	}
	if info.Token == "" || info.URI == "" {
		t.Fatalf("expected token and uri: %+v", info)
	}
	if _, err := store.Pairings.Get(info.Token); err != nil {
		t.Fatalf("token should be stored: %v", err)
	}
}
