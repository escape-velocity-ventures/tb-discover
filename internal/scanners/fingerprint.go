package scanners

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ComputeHardwareID generates a stable fingerprint from hardware identifiers.
// Uses sorted MAC addresses and disk serial numbers to produce a hash that
// is identical regardless of reporting source (standalone vs DaemonSet).
func ComputeHardwareID(interfaces []NetworkInterface, storage []StorageDevice) string {
	var parts []string

	// Collect MAC addresses (skip empty, virtual interfaces tend to have random MACs)
	for _, iface := range interfaces {
		if iface.MAC == "" {
			continue
		}
		// Only use physical interfaces for fingerprinting
		if iface.Type == "ethernet" || iface.Type == "wifi" {
			parts = append(parts, "mac:"+strings.ToLower(iface.MAC))
		}
	}

	// Collect disk serial numbers
	for _, dev := range storage {
		if dev.Serial != "" {
			parts = append(parts, "serial:"+dev.Serial)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Sort for deterministic output
	sort.Strings(parts)

	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", hash[:12]) // 24-char hex = 96 bits, plenty for correlation
}
