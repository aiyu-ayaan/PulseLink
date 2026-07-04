package auth

import (
	"testing"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
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

func TestUnknownAndroidDeviceRequestsPairing(t *testing.T) {
	store := newTestStore(t)
	bus := eventbus.New()
	auth := New(store, bus)

	var requests int
	bus.Subscribe("pairing.request", func(ev eventbus.Event) {
		requests++
	})

	res, err := auth.Authenticate(protocol.ClientHello{
		DeviceID:     "android-pixel",
		DeviceName:   "Pixel",
		Token:        "desktop-local",
		Capabilities: []string{"pairing", "volume", "sysinfo"},
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !res.Accepted {
		t.Fatalf("expected accepted pending session, got rejected: %s", res.Reason)
	}
	if got := res.AllowedCapabilities; len(got) != 1 || got[0] != "pairing" {
		t.Fatalf("pending device should only receive pairing capability, got %v", got)
	}
	if requests != 1 {
		t.Fatalf("expected one pairing request event, got %d", requests)
	}

	dev, err := store.Devices.Get("android-pixel")
	if err != nil {
		t.Fatalf("device stored: %v", err)
	}
	if dev.Trusted {
		t.Fatal("new device should be pending until accepted")
	}
}

func TestTrustedAndroidDeviceGetsFullNegotiation(t *testing.T) {
	store := newTestStore(t)
	bus := eventbus.New()
	auth := New(store, bus)

	if err := store.Devices.Upsert(storage.Device{
		ID:           "android-pixel",
		Name:         "Pixel",
		PublicKey:    "desktop-local",
		Trusted:      true,
		Capabilities: []string{"pairing", "volume", "sysinfo"},
	}); err != nil {
		t.Fatalf("seed device: %v", err)
	}

	res, err := auth.Authenticate(protocol.ClientHello{
		DeviceID:     "android-pixel",
		DeviceName:   "Pixel",
		Capabilities: []string{"pairing", "volume", "sysinfo"},
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !res.Accepted {
		t.Fatalf("expected accepted trusted session, got rejected: %s", res.Reason)
	}
	if res.AllowedCapabilities != nil {
		t.Fatalf("trusted device should not be narrowed by auth, got %v", res.AllowedCapabilities)
	}
}
