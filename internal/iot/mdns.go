package iot

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// MDNSProvider discovers IoT devices via mDNS/Bonjour.
type MDNSProvider struct{}

func NewMDNSProvider() *MDNSProvider { return &MDNSProvider{} }

func (p *MDNSProvider) Name() string { return "mdns" }

func (p *MDNSProvider) Detect(ctx context.Context) (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		// dns-sd is always available on macOS
		_, err := exec.LookPath("dns-sd")
		return err == nil, nil
	case "linux":
		_, err := exec.LookPath("avahi-browse")
		return err == nil, nil
	}
	return false, nil
}

func (p *MDNSProvider) Discover(ctx context.Context) ([]Device, error) {
	switch runtime.GOOS {
	case "darwin":
		return p.discoverDarwin(ctx)
	case "linux":
		return p.discoverLinux(ctx)
	}
	return nil, nil
}

// Service types that indicate IoT devices.
var mdnsServiceTypes = []struct {
	service    string
	deviceType DeviceType
	label      string
}{
	{"_hap._tcp", TypeUnknown, "HomeKit"},
	{"_hue._tcp", TypeLight, "Hue Bridge"},
	{"_airplay._tcp", TypeMedia, "AirPlay"},
	{"_googlecast._tcp", TypeMedia, "Chromecast"},
	{"_raop._tcp", TypeMedia, "AirPlay Audio"},
	{"_homekit._tcp", TypeUnknown, "HomeKit"},
	{"_smartenergy._tcp", TypeAppliance, "Smart Energy"},
}

func (p *MDNSProvider) discoverDarwin(ctx context.Context) ([]Device, error) {
	var devices []Device

	for _, svc := range mdnsServiceTypes {
		// dns-sd -B with a short timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cmd := exec.CommandContext(timeoutCtx, "dns-sd", "-B", svc.service, "local.")
		out, _ := cmd.CombinedOutput()
		cancel()

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "DATE:") || strings.HasPrefix(line, "Browsing") || strings.HasPrefix(line, "Timestamp") {
				continue
			}

			// Parse dns-sd output: Timestamp A/R Flags if Domain Service Type Instance Name
			fields := strings.Fields(line)
			if len(fields) < 7 {
				continue
			}

			// Instance name is everything after the 6th field
			name := strings.Join(fields[6:], " ")
			if name == "" {
				continue
			}

			devices = append(devices, Device{
				ID:     fmt.Sprintf("mdns-%s-%s", svc.service, sanitizeMDNSName(name)),
				Name:   name,
				Type:   svc.deviceType,
				State:  "discovered",
				Source: "mdns",
				Attributes: map[string]interface{}{
					"service": svc.service,
					"label":   svc.label,
				},
			})
		}
	}

	return dedup(devices), nil
}

func (p *MDNSProvider) discoverLinux(ctx context.Context) ([]Device, error) {
	var devices []Device

	for _, svc := range mdnsServiceTypes {
		timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cmd := exec.CommandContext(timeoutCtx, "avahi-browse", "-tpr", svc.service)
		out, _ := cmd.CombinedOutput()
		cancel()

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "=") {
				continue
			}

			// avahi-browse output: =;iface;protocol;name;type;domain;host;address;port;txt
			fields := strings.Split(line, ";")
			if len(fields) < 8 {
				continue
			}

			name := fields[3]
			if name == "" {
				continue
			}

			devices = append(devices, Device{
				ID:     fmt.Sprintf("mdns-%s-%s", svc.service, sanitizeMDNSName(name)),
				Name:   name,
				Type:   svc.deviceType,
				State:  "discovered",
				Source: "mdns",
				Attributes: map[string]interface{}{
					"service": svc.service,
					"label":   svc.label,
					"host":    fields[6],
					"address": fields[7],
				},
			})
		}
	}

	return dedup(devices), nil
}

func sanitizeMDNSName(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(s))
}

func dedup(devices []Device) []Device {
	seen := make(map[string]bool)
	var result []Device
	for _, d := range devices {
		if !seen[d.ID] {
			seen[d.ID] = true
			result = append(result, d)
		}
	}
	return result
}

