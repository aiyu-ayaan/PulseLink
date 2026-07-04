package auth

import (
	"testing"
	"time"

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

func TestUnknownDeviceWithValidPairingTokenRequestsPairing(t *testing.T) {
	store := newTestStore(t)
	bus := eventbus.New()
	auth := New(store, bus)
	now := time.Now()

	if err := store.Pairings.Create(storage.Pairing{
		Token:     "valid-token",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("seed pairing token: %v", err)
	}

	res, err := auth.Authenticate(protocol.ClientHello{
		DeviceID:     "android-pixel",
		DeviceName:   "Pixel",
		Token:        "valid-token",
		Capabilities: []string{"pairing", "volume"},
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !res.Accepted {
		t.Fatalf("expected valid token to be accepted pending approval: %s", res.Reason)
	}

	pairing, err := store.Pairings.Get("valid-token")
	if err != nil {
		t.Fatalf("get pairing token: %v", err)
	}
	if !pairing.Used || pairing.DeviceID != "android-pixel" {
		t.Fatalf("token should be marked used by device, got %+v", pairing)
	}
}

func TestUnknownDeviceRejectsInvalidPairingToken(t *testing.T) {
	store := newTestStore(t)
	auth := New(store, eventbus.New())

	res, err := auth.Authenticate(protocol.ClientHello{
		DeviceID:     "android-pixel",
		DeviceName:   "Pixel",
		Token:        "not-real",
		Capabilities: []string{"pairing", "volume"},
	})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if res.Accepted || res.Reason != "invalid pairing token" {
		t.Fatalf("expected invalid token rejection, got %+v", res)
	}
}
