package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all tb-discover configuration.
type Config struct {
	Token        string        `yaml:"token"`
	URL          string        `yaml:"url"`
	Profile      string        `yaml:"profile"`
	ScanInterval time.Duration `yaml:"scan_interval"`
	LogLevel     string        `yaml:"log_level"`
	Permissions  []string      `yaml:"permissions"` // e.g., ["terminal", "scan"]
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Profile:      "standard",
		ScanInterval: 5 * time.Minute,
		LogLevel:     "info",
		Permissions:  []string{"scan"},
	}
}

// Load reads a YAML config file and overlays environment variables.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Environment variable overrides
	if v := os.Getenv("TB_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("TB_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("TB_PROFILE"); v != "" {
		cfg.Profile = v
	}
	if v := os.Getenv("TB_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg, nil
}
