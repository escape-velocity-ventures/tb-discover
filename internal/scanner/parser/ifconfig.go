package parser

import (
	"strconv"
	"strings"
)

// InterfaceInfo matches the scanner.InterfaceInfo shape for parser output.
type InterfaceInfo struct {
	Name  string `json:"name"`
	IP    string `json:"ip,omitempty"`
	IPv6  string `json:"ipv6,omitempty"`
	MAC   string `json:"mac,omitempty"`
	MTU   int    `json:"mtu,omitempty"`
	State string `json:"state,omitempty"`
	Type  string `json:"type,omitempty"`
}

// ParseIfconfig parses macOS/BSD ifconfig -a output.
func ParseIfconfig(output string) []InterfaceInfo {
	var interfaces []InterfaceInfo
	var current *InterfaceInfo

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		// New interface starts at column 0 (no leading whitespace)
		if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
			if current != nil {
				interfaces = append(interfaces, *current)
			}
			current = &InterfaceInfo{}

			// Parse "en0: flags=8863<UP,...> mtu 1500"
			colonIdx := strings.Index(line, ":")
			if colonIdx > 0 {
				current.Name = line[:colonIdx]
			}

			// Extract MTU
			if idx := strings.Index(line, "mtu "); idx >= 0 {
				rest := line[idx+4:]
				fields := strings.Fields(rest)
				if len(fields) > 0 {
					if mtu, err := strconv.Atoi(fields[0]); err == nil {
						current.MTU = mtu
					}
				}
			}

			// Extract UP/DOWN from flags
			if strings.Contains(line, "<UP") || strings.Contains(line, ",UP") || strings.Contains(line, "<UP,") {
				current.State = "up"
			} else {
				current.State = "down"
			}

			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// "ether aa:bb:cc:dd:ee:ff"
		if strings.HasPrefix(trimmed, "ether ") {
			current.MAC = strings.Fields(trimmed)[1]
		}

		// "inet 192.168.1.10 netmask ..."
		if strings.HasPrefix(trimmed, "inet ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				current.IP = fields[1]
			}
		}

		// "inet6 fe80::1%en0 prefixlen 64 scopeid 0x4"
		if strings.HasPrefix(trimmed, "inet6 ") && current.IPv6 == "" {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				addr := fields[1]
				// Strip %interface suffix
				if idx := strings.Index(addr, "%"); idx >= 0 {
					addr = addr[:idx]
				}
				// Skip link-local addresses for primary IPv6
				if !strings.HasPrefix(addr, "fe80") {
					current.IPv6 = addr
				}
			}
		}
	}

	if current != nil {
		interfaces = append(interfaces, *current)
	}

	return interfaces
}
