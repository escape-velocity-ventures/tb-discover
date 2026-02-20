package iot

import (
	"context"
	"log/slog"
)

// Registry manages IoT providers and auto-detects available ones.
type Registry struct {
	all []Provider
	log *slog.Logger
}

// NewRegistry creates a registry with all known IoT providers.
func NewRegistry() *Registry {
	return &Registry{
		all: []Provider{
			NewHomeAssistantProvider(),
			NewMDNSProvider(),
			NewHueProvider(),
			NewUniFiProvider(),
		},
		log: slog.Default().With("component", "iot"),
	}
}

// Scan detects providers and discovers all IoT devices.
func (r *Registry) Scan(ctx context.Context) DiscoveryResult {
	result := DiscoveryResult{}

	for _, p := range r.all {
		ok, err := p.Detect(ctx)
		if err != nil {
			r.log.Debug("iot provider detection failed", "provider", p.Name(), "error", err)
			continue
		}
		if !ok {
			continue
		}

		r.log.Info("iot provider detected", "provider", p.Name())
		result.Providers = append(result.Providers, p.Name())

		devices, err := p.Discover(ctx)
		if err != nil {
			r.log.Warn("iot discovery failed", "provider", p.Name(), "error", err)
			continue
		}
		result.Devices = append(result.Devices, devices...)
	}

	return result
}
