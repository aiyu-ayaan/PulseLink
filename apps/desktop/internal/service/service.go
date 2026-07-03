// Package service defines the contract every feature module implements and a
// registry that manages their lifecycle.
package service

import (
	"context"
	"fmt"
	"log/slog"
)

// Service is one isolated feature module (media, brightness, clipboard, ...).
//
// Modules are self-contained: they receive their dependencies at construction
// and communicate with the rest of the system only through the event bus.
type Service interface {
	// Name is a stable identifier, also used as the capability name.
	Name() string
	// Start begins any background work. It must return promptly; long-running
	// loops belong in goroutines that stop when ctx is cancelled.
	Start(ctx context.Context) error
	// Stop releases resources. It should be idempotent.
	Stop(ctx context.Context) error
}

// Registry owns a set of services and starts/stops them as a group.
type Registry struct {
	log     *slog.Logger
	svcs    []Service
	started []Service
}

// NewRegistry creates an empty registry.
func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{log: log}
}

// Register adds a service. Order matters: services start in registration order
// and stop in reverse.
func (r *Registry) Register(s Service) {
	r.svcs = append(r.svcs, s)
}

// Names returns the registered service names (capabilities).
func (r *Registry) Names() []string {
	names := make([]string, len(r.svcs))
	for i, s := range r.svcs {
		names[i] = s.Name()
	}
	return names
}

// StartAll starts every service. If one fails, already-started services are
// stopped and the error is returned.
func (r *Registry) StartAll(ctx context.Context) error {
	for _, s := range r.svcs {
		r.log.Info("starting service", "service", s.Name())
		if err := s.Start(ctx); err != nil {
			_ = r.StopAll(ctx)
			return fmt.Errorf("start %s: %w", s.Name(), err)
		}
		r.started = append(r.started, s)
	}
	return nil
}

// StopAll stops started services in reverse order, collecting errors.
func (r *Registry) StopAll(ctx context.Context) error {
	var firstErr error
	for i := len(r.started) - 1; i >= 0; i-- {
		s := r.started[i]
		r.log.Info("stopping service", "service", s.Name())
		if err := s.Stop(ctx); err != nil {
			r.log.Error("service stop failed", "service", s.Name(), "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	r.started = nil
	return firstErr
}
