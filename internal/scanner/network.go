package scanner

import (
	"context"
	"encoding/json"
)

// NetworkInfo holds all network data from a scan.
type NetworkInfo struct {
	Hostname   string          `json:"hostname"`
	Interfaces []InterfaceInfo `json:"interfaces"`
	Routes     []RouteInfo     `json:"routes,omitempty"`
}

// InterfaceInfo represents a single network interface.
type InterfaceInfo struct {
	Name  string `json:"name"`
	IP    string `json:"ip,omitempty"`
	IPv6  string `json:"ipv6,omitempty"`
	MAC   string `json:"mac,omitempty"`
	MTU   int    `json:"mtu,omitempty"`
	State string `json:"state,omitempty"` // up, down
	Type  string `json:"type,omitempty"`  // physical, cni, bridge, virtual, tunnel, wireless
}

// RouteInfo represents a network route.
type RouteInfo struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Metric      int    `json:"metric,omitempty"`
}

// NetworkScanner collects network interface and routing information.
type NetworkScanner struct{}

// NewNetworkScanner creates a new NetworkScanner.
func NewNetworkScanner() *NetworkScanner {
	return &NetworkScanner{}
}

func (s *NetworkScanner) Name() string       { return "network" }
func (s *NetworkScanner) Platforms() []string { return nil }

func (s *NetworkScanner) Scan(ctx context.Context, runner CommandRunner) (json.RawMessage, error) {
	info := NetworkInfo{}

	// Hostname
	if out, err := runner.Run(ctx, "hostname"); err == nil {
		info.Hostname = trimOutput(out)
	}

	// Collect platform-specific interface and route data
	if err := collectNetworkInfo(ctx, runner, &info); err != nil {
		return nil, err
	}

	return json.Marshal(info)
}
