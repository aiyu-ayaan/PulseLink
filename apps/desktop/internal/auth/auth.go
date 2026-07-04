package auth

import (
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
	if hello.DeviceID == "desktop-ui" {
		return transport.AuthResult{
			Accepted:   true,
			DeviceID:   hello.DeviceID,
			DeviceName: hello.DeviceName,
		}, nil
	}

	dev, err := a.store.Devices.Get(hello.DeviceID)
	if err == nil {
		if dev.Trusted {
			_ = a.store.Devices.TouchLastSeen(hello.DeviceID, time.Now())
			return transport.AuthResult{
				Accepted:   true,
				DeviceID:   hello.DeviceID,
				DeviceName: hello.DeviceName,
			}, nil
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

	if !errors.Is(err, storage.ErrNotFound) {
		return transport.AuthResult{}, err
	}

	publicKey := hello.Token
	if publicKey == "" {
		publicKey = "manual-pairing"
	}

	d := storage.Device{
		ID:           hello.DeviceID,
		Name:         hello.DeviceName,
		PublicKey:    publicKey,
		Trusted:      false,
		PairedAt:     time.Now(),
		LastSeen:     time.Now(),
		Capabilities: hello.Capabilities,
	}
	if err := a.store.Devices.Upsert(d); err != nil {
		return transport.AuthResult{}, err
	}

	if hello.Token != "" && hello.Token != "desktop-local" {
		_ = a.store.Pairings.MarkUsed(hello.Token, hello.DeviceID)
	}

	info := transport.DeviceInfo{
		ID:           hello.DeviceID,
		Name:         hello.DeviceName,
		Capabilities: hello.Capabilities,
	}

	a.bus.Publish(eventbus.Event{
		Topic:   "pairing.request",
		Payload: info,
	})

	return transport.AuthResult{
		Accepted:            true,
		DeviceID:            hello.DeviceID,
		DeviceName:          hello.DeviceName,
		AllowedCapabilities: []string{"pairing"},
	}, nil
}
