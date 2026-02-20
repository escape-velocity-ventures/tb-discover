package scanner

import (
	"context"
	"encoding/json"
	"strings"
)

// ContainerInfo holds container runtime discovery results.
type ContainerInfo struct {
	Runtime    string          `json:"runtime,omitempty"` // docker, podman, containerd, nerdctl
	Version    string          `json:"version,omitempty"`
	Containers []ContainerItem `json:"containers,omitempty"`
	Images     []string        `json:"images,omitempty"`
}

// ContainerItem represents a running container.
type ContainerItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	State string `json:"state"`
}

// ContainerScanner detects container runtimes and lists containers.
type ContainerScanner struct{}

// NewContainerScanner creates a new ContainerScanner.
func NewContainerScanner() *ContainerScanner {
	return &ContainerScanner{}
}

func (s *ContainerScanner) Name() string       { return "containers" }
func (s *ContainerScanner) Platforms() []string { return nil }

func (s *ContainerScanner) Scan(ctx context.Context, runner CommandRunner) (json.RawMessage, error) {
	info := ContainerInfo{}

	// Try runtimes in priority order
	runtimes := []struct {
		name    string
		check   string
		version string
		ps      string
		images  string
	}{
		{"docker", "docker info", "docker version --format '{{.Server.Version}}'", "docker ps --format '{{.ID}}\\t{{.Names}}\\t{{.Image}}\\t{{.State}}'", "docker images --format '{{.Repository}}:{{.Tag}}'"},
		{"podman", "podman info", "podman version --format '{{.Version}}'", "podman ps --format '{{.ID}}\\t{{.Names}}\\t{{.Image}}\\t{{.State}}'", "podman images --format '{{.Repository}}:{{.Tag}}'"},
		{"nerdctl", "nerdctl info", "nerdctl version --format '{{.Client.Version}}'", "nerdctl ps --format '{{.ID}}\\t{{.Names}}\\t{{.Image}}\\t{{.Status}}'", "nerdctl images --format '{{.Repository}}:{{.Tag}}'"},
	}

	for _, rt := range runtimes {
		if _, err := runner.Run(ctx, rt.check+" 2>/dev/null"); err != nil {
			continue
		}

		info.Runtime = rt.name

		// Version
		if out, err := runner.Run(ctx, rt.version+" 2>/dev/null"); err == nil {
			info.Version = trimOutput(out)
		}

		// Running containers
		if out, err := runner.Run(ctx, rt.ps+" 2>/dev/null"); err == nil {
			info.Containers = parseContainerPS(string(out))
		}

		// Images
		if out, err := runner.Run(ctx, rt.images+" 2>/dev/null"); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && line != "<none>:<none>" {
					info.Images = append(info.Images, line)
				}
			}
		}

		break // Use first available runtime
	}

	return json.Marshal(info)
}

func parseContainerPS(output string) []ContainerItem {
	var containers []ContainerItem
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		item := ContainerItem{
			ID:    parts[0],
			Name:  parts[1],
			Image: parts[2],
		}
		if len(parts) >= 4 {
			item.State = parts[3]
		}
		containers = append(containers, item)
	}
	return containers
}
