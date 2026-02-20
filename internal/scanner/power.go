package scanner

import (
	"context"
	"encoding/json"

	"github.com/tinkerbelle-io/tb-discover/internal/power"
)

// PowerScanner detects available power control mechanisms.
type PowerScanner struct{}

func NewPowerScanner() *PowerScanner { return &PowerScanner{} }

func (s *PowerScanner) Name() string       { return "power" }
func (s *PowerScanner) Platforms() []string { return nil }

func (s *PowerScanner) Scan(ctx context.Context, _ CommandRunner) (json.RawMessage, error) {
	reg := power.NewRegistry()
	caps := reg.Scan(ctx)
	return json.Marshal(caps)
}
