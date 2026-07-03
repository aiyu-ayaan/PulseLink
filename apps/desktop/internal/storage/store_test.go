package storage

import (
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDeviceRoundTrip(t *testing.T) {
	s := newTestStore(t)
	now := time.Unix(1000, 0)
	d := Device{
		ID:           "dev-1",
		Name:         "Pixel",
		Trusted:      true,
		PairedAt:     now,
		LastSeen:     now,
		Capabilities: []string{"media", "brightness"},
	}
	if err := s.Devices.Upsert(d); err != nil {
		t.Fatal(err)
	}
	got, err := s.Devices.Get("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Pixel" || !got.Trusted || len(got.Capabilities) != 2 {
		t.Fatalf("unexpected device: %+v", got)
	}

	if err := s.Devices.Delete("dev-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Devices.Get("dev-1"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSettings(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Settings.Get("missing"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := s.Settings.Set("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	v, err := s.Settings.Get("theme")
	if err != nil || v != "dark" {
		t.Fatalf("got %q, %v", v, err)
	}
}

func TestPairingExpiry(t *testing.T) {
	s := newTestStore(t)
	past := time.Unix(500, 0)
	_ = s.Pairings.Create(Pairing{Token: "t1", CreatedAt: past, ExpiresAt: past})
	n, err := s.Pairings.DeleteExpired(time.Unix(1000, 0))
	if err != nil || n != 1 {
		t.Fatalf("want 1 deleted, got %d (%v)", n, err)
	}
}
