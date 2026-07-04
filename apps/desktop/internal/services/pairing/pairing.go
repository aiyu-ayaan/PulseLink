package pairing

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

type Service struct {
	log   *slog.Logger
	bus   *eventbus.Bus
	store *storage.Store
	hub   *transport.Hub
	cfg   config.Config
}

func New(log *slog.Logger, bus *eventbus.Bus, store *storage.Store, hub *transport.Hub, cfg config.Config) *Service {
	return &Service{
		log:   log,
		bus:   bus,
		store: store,
		hub:   hub,
		cfg:   cfg,
	}
}

func (s *Service) Name() string {
	return "pairing"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("pairing service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("pairing service stopping")
	return nil
}

type DeviceActionPayload struct {
	DeviceID string `json:"deviceId"`
}

type Info struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Scheme    string `json:"scheme"`
	Token     string `json:"token"`
	URI       string `json:"uri"`
	ExpiresAt int64  `json:"expiresAt"`
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "info":
		return s.newPairingInfo()

	case "list":
		all, err := s.store.Devices.List()
		if err != nil {
			return nil, err
		}
		var pending []transport.DeviceInfo
		for _, d := range all {
			if !d.Trusted {
				pending = append(pending, transport.DeviceInfo{
					ID:           d.ID,
					Name:         d.Name,
					Capabilities: d.Capabilities,
				})
			}
		}
		return pending, nil

	case "pending":
		all, err := s.store.Devices.List()
		if err != nil {
			return nil, err
		}
		count := 0
		var pending []transport.DeviceInfo
		for _, d := range all {
			if !d.Trusted {
				count++
				pending = append(pending, transport.DeviceInfo{
					ID:           d.ID,
					Name:         d.Name,
					Capabilities: d.Capabilities,
				})
			}
		}
		return map[string]any{
			"count":   count,
			"devices": pending,
		}, nil

	case "accept":
		var payload DeviceActionPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed pairing payload"}
		}
		payload.DeviceID = strings.TrimSpace(payload.DeviceID)
		if payload.DeviceID == "" {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "deviceId is required"}
		}

		s.log.Info("pairing accepted", "device", payload.DeviceID)
		if err := s.store.Devices.SetTrusted(payload.DeviceID, true); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, &protocol.Error{Code: protocol.CodeNotFound, Message: "pairing request not found"}
			}
			return nil, err
		}

		s.bus.Publish(eventbus.Event{
			Topic:   "pairing.approved",
			Payload: payload.DeviceID,
		})
		return map[string]string{"status": "approved", "deviceId": payload.DeviceID}, nil

	case "reject":
		var payload DeviceActionPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed pairing payload"}
		}
		payload.DeviceID = strings.TrimSpace(payload.DeviceID)
		if payload.DeviceID == "" {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "deviceId is required"}
		}

		s.log.Info("pairing rejected", "device", payload.DeviceID)
		_ = s.store.Devices.Delete(payload.DeviceID)

		s.bus.Publish(eventbus.Event{
			Topic:   "pairing.rejected",
			Payload: payload.DeviceID,
		})
		return map[string]string{"status": "rejected", "deviceId": payload.DeviceID}, nil

	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown pairing action"}
	}
}

func (s *Service) newPairingInfo() (Info, error) {
	token, err := randomToken()
	if err != nil {
		return Info{}, err
	}
	now := time.Now()
	expires := now.Add(10 * time.Minute)
	if _, err := s.store.Pairings.DeleteExpired(now); err != nil {
		return Info{}, err
	}
	if err := s.store.Pairings.Create(storage.Pairing{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: expires,
	}); err != nil {
		return Info{}, err
	}

	host := pairingHost(s.cfg.Server.Host)
	scheme := "ws"
	if s.cfg.Server.EnableTLS {
		scheme = "wss"
	}

	q := url.Values{}
	q.Set("host", host)
	q.Set("port", strconv.Itoa(s.cfg.Server.Port))
	q.Set("token", token)
	q.Set("name", s.cfg.DeviceName)
	q.Set("scheme", scheme)

	return Info{
		Host:      host,
		Port:      s.cfg.Server.Port,
		Scheme:    scheme,
		Token:     token,
		URI:       "pulselink://pair?" + q.Encode(),
		ExpiresAt: expires.Unix(),
	}, nil
}

func randomToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate pairing token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func pairingHost(configured string) string {
	host := strings.TrimSpace(configured)
	if host != "" && host != "0.0.0.0" && host != "::" && host != "[::]" {
		if parsed := net.ParseIP(host); parsed == nil || !parsed.IsUnspecified() {
			return host
		}
	}

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ip, ok := addrIP(addr)
				if !ok || ip.IsLoopback() || ip.IsUnspecified() {
					continue
				}
				if v4 := ip.To4(); v4 != nil {
					return v4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func addrIP(addr net.Addr) (net.IP, bool) {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP, true
	case *net.IPAddr:
		return v.IP, true
	default:
		return nil, false
	}
}
