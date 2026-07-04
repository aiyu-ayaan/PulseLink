package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

type Authenticator struct {
	store *storage.Store
	bus   *eventbus.Bus
}

func New(store *storage.Store, bus *eventbus.Bus) *Authenticator {
	return &Authenticator{
		store: store,
		bus:   bus,
	}
}

func (a *Authenticator) Authenticate(hello protocol.ClientHello) (transport.AuthResult, error) {
	// If the connection is from the desktop-ui itself, allow it
	if hello.DeviceID == "desktop-ui" {
		return transport.AuthResult{
			Accepted:   true,
			DeviceID:   hello.DeviceID,
			DeviceName: hello.DeviceName,
		}, nil
	}

	// Check if this device is already in the database
	dev, err := a.store.Devices.Get(hello.DeviceID)
	if err == nil {
		if dev.Trusted {
			// Update last seen
			_ = a.store.Devices.TouchLastSeen(hello.DeviceID, time.Now())
			return transport.AuthResult{
				Accepted:   true,
				DeviceID:   hello.DeviceID,
				DeviceName: hello.DeviceName,
			}, nil
		}

		// Device exists but is not yet trusted. Trigger pairing notification again.
		a.bus.Publish(eventbus.Event{
			Topic: "pairing.request",
			Payload: transport.DeviceInfo{
				ID:           hello.DeviceID,
				Name:         hello.DeviceName,
				Capabilities: hello.Capabilities,
			},
		})

		return transport.AuthResult{
			Accepted:            true,
			DeviceID:            hello.DeviceID,
			DeviceName:          hello.DeviceName,
			AllowedCapabilities: []string{"pairing"},
		}, nil
	}

	if !errors.Is(err, storage.ErrNotFound) {
		return transport.AuthResult{}, err
	}

	// Check pairing token
	pair, err := a.store.Pairings.Get(hello.Token)
	if err == nil && !pair.Used && time.Now().Before(pair.ExpiresAt) {
		d := storage.Device{
			ID:           hello.DeviceID,
			Name:         hello.DeviceName,
			PublicKey:    hello.Token,
			Trusted:      false,
			PairedAt:     time.Now(),
			LastSeen:     time.Now(),
			Capabilities: hello.Capabilities,
		}
		if err := a.store.Devices.Upsert(d); err != nil {
			return transport.AuthResult{}, err
		}
		
		_ = a.store.Pairings.MarkUsed(hello.Token, hello.DeviceID)

		a.bus.Publish(eventbus.Event{
			Topic: "pairing.request",
			Payload: transport.DeviceInfo{
				ID:           hello.DeviceID,
				Name:         hello.DeviceName,
				Capabilities: hello.Capabilities,
			},
		})

		return transport.AuthResult{
			Accepted:            true,
			DeviceID:            hello.DeviceID,
			DeviceName:          hello.DeviceName,
			AllowedCapabilities: []string{"pairing"},
		}, nil
	}

	// For dev / fallback mode
	if hello.Token == "desktop-local" {
		d := storage.Device{
			ID:           hello.DeviceID,
			Name:         hello.DeviceName,
			PublicKey:    "desktop-local",
			Trusted:      false,
			PairedAt:     time.Now(),
			LastSeen:     time.Now(),
			Capabilities: hello.Capabilities,
		}
		if err := a.store.Devices.Upsert(d); err != nil {
			return transport.AuthResult{}, err
		}

		a.bus.Publish(eventbus.Event{
			Topic: "pairing.request",
			Payload: transport.DeviceInfo{
				ID:           hello.DeviceID,
				Name:         hello.DeviceName,
				Capabilities: hello.Capabilities,
			},
		})

		return transport.AuthResult{
			Accepted:            true,
			DeviceID:            hello.DeviceID,
			DeviceName:          hello.DeviceName,
			AllowedCapabilities: []string{"pairing"},
		}, nil
	}

	return transport.AuthResult{
		Accepted: false,
		Reason:   "pairing token invalid or expired",
	}, nil
}
