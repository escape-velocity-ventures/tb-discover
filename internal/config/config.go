// Package config handles configuration for tb-discover.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all tb-discover configuration.
type Config struct {
	AgentToken          string
	IngestURL           string
	AnonKey             string
	ScanIntervalSeconds int
	NodeName            string
	Mode                string // "daemon", "oneshot", "k8s"
	Profile             string // "minimal", "standard", "full"
	HostType            string // "baremetal", "vm", "cloud"
}

// Load reads configuration from environment variables.
// Config file support (YAML) will be added in a future phase.
func Load() (*Config, error) {
	token := os.Getenv("AGENT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("AGENT_TOKEN environment variable is required")
	}

	ingestURL := os.Getenv("INGEST_URL")
	if ingestURL == "" {
		return nil, fmt.Errorf("INGEST_URL environment variable is required")
	}

	anonKey := os.Getenv("ANON_KEY")
	if anonKey == "" {
		return nil, fmt.Errorf("ANON_KEY environment variable is required")
	}

	interval := 1800
	if v := os.Getenv("SCAN_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			interval = n
		}
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "auto"
	}

	mode := os.Getenv("DISCOVER_MODE")
	if mode == "" {
		mode = "daemon"
	}

	profile := os.Getenv("SCAN_PROFILE")
	if profile == "" {
		profile = "full"
	}

	hostType := os.Getenv("HOST_TYPE")
	if hostType == "" {
		hostType = "baremetal"
	}

	return &Config{
		AgentToken:          token,
		IngestURL:           ingestURL,
		AnonKey:             anonKey,
		ScanIntervalSeconds: interval,
		NodeName:            nodeName,
		Mode:                mode,
		Profile:             profile,
		HostType:            hostType,
	}, nil
}
