package power

import (
	"context"
	"log/slog"
)

// Registry manages power providers and auto-detects available ones.
type Registry struct {
	all       []Provider
	available []Provider
	log       *slog.Logger
}

// NewRegistry creates a registry with all known providers.
func NewRegistry() *Registry {
	return &Registry{
		all: []Provider{
			NewIPMIProvider(),
			NewWoLProvider(),
			NewHypervisorProvider(),
			NewSmartPlugProvider(),
			NewPoEProvider(),
			NewCloudProvider(),
		},
		log: slog.Default().With("component", "power"),
	}
}

// Detect probes all providers and returns those that are available.
func (r *Registry) Detect(ctx context.Context) []Provider {
	r.available = nil

	for _, p := range r.all {
		ok, err := p.Detect(ctx)
		if err != nil {
			r.log.Debug("provider detection failed", "provider", p.Name(), "error", err)
			continue
		}
		if ok {
			r.log.Info("power provider detected", "provider", p.Name())
			r.available = append(r.available, p)
		}
	}

	return r.available
}

// Available returns providers that passed detection.
func (r *Registry) Available() []Provider {
	return r.available
}

// Scan detects providers, lists all targets, and builds capabilities.
func (r *Registry) Scan(ctx context.Context) PowerCapabilities {
	providers := r.Detect(ctx)

	caps := PowerCapabilities{}

	for _, p := range providers {
		caps.Providers = append(caps.Providers, p.Name())

		targets, err := p.ListTargets(ctx)
		if err != nil {
			r.log.Warn("failed to list targets", "provider", p.Name(), "error", err)
			continue
		}
		caps.Targets = append(caps.Targets, targets...)
	}

	return caps
}
