package scanner

import (
	"context"
	"encoding/json"

	"github.com/tinkerbelle-io/tb-discover/internal/iot"
)

// IoTScanner discovers IoT devices via available providers.
type IoTScanner struct{}

func NewIoTScanner() *IoTScanner { return &IoTScanner{} }

func (s *IoTScanner) Name() string       { return "iot" }
func (s *IoTScanner) Platforms() []string { return nil }

func (s *IoTScanner) Scan(ctx context.Context, _ CommandRunner) (json.RawMessage, error) {
	reg := iot.NewRegistry()
	result := reg.Scan(ctx)
	return json.Marshal(result)
}
