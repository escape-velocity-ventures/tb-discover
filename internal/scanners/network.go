package scanners

import (
	"regexp"
	"runtime"
	"strings"
)

// ScanNetwork discovers network interfaces and their addresses.
func ScanNetwork() []NetworkInterface {
	switch runtime.GOOS {
	case "linux":
		return scanLinuxNetwork()
	case "darwin":
		return scanDarwinNetwork()
	default:
		return nil
	}
}

func scanLinuxNetwork() []NetworkInterface {
	result := HostExec("ip -o addr show 2>/dev/null")
	if result.ExitCode != 0 {
		return nil
	}
	return parseIPAddr(result.Stdout)
}

// parseIPAddr parses `ip -o addr show` output.
func parseIPAddr(output string) []NetworkInterface {
	ifaceMap := make(map[string]*NetworkInterface)
	var order []string

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := strings.TrimSuffix(fields[1], ":")
		family := fields[2] // "inet" or "inet6"
		addrField := fields[3]

		// Strip CIDR suffix
		addr := strings.Split(addrField, "/")[0]

		iface, exists := ifaceMap[name]
		if !exists {
			iface = &NetworkInterface{
				Name:   name,
				Type:   guessInterfaceType(name),
				Status: "up",
			}
			ifaceMap[name] = iface
			order = append(order, name)
		}

		switch family {
		case "inet":
			if iface.IP == "" {
				iface.IP = addr
			}
		case "inet6":
			// Skip link-local
			if !strings.HasPrefix(addr, "fe80:") && iface.IPv6 == "" {
				iface.IPv6 = addr
			}
		}
	}

	// Get MAC addresses
	macResult := HostExec("ip -o link show 2>/dev/null")
	if macResult.ExitCode == 0 {
		macRe := regexp.MustCompile(`link/\w+\s+([0-9a-f:]{17})`)
		for _, line := range strings.Split(macResult.Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			name := strings.TrimSuffix(fields[1], ":")
			if iface, ok := ifaceMap[name]; ok {
				if m := macRe.FindStringSubmatch(line); len(m) > 1 {
					iface.MAC = m[1]
				}
			}
		}
	}

	var interfaces []NetworkInterface
	for _, name := range order {
		// Skip loopback
		if name == "lo" {
			continue
		}
		interfaces = append(interfaces, *ifaceMap[name])
	}

	return interfaces
}

func scanDarwinNetwork() []NetworkInterface {
	result := HostExec("ifconfig 2>/dev/null")
	if result.ExitCode != 0 {
		return nil
	}

	var interfaces []NetworkInterface
	var current *NetworkInterface

	for _, line := range strings.Split(result.Stdout, "\n") {
		if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
			// New interface
			if current != nil && current.Name != "lo0" {
				interfaces = append(interfaces, *current)
			}
			name := strings.TrimSuffix(strings.Fields(line)[0], ":")
			current = &NetworkInterface{
				Name:   name,
				Type:   guessInterfaceType(name),
				Status: "up",
			}
			if strings.Contains(line, "status: inactive") {
				current.Status = "down"
			}
		} else if current != nil {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "inet ") {
				fields := strings.Fields(trimmed)
				if len(fields) >= 2 {
					current.IP = fields[1]
				}
			}
			if strings.HasPrefix(trimmed, "inet6 ") && !strings.Contains(trimmed, "fe80") {
				fields := strings.Fields(trimmed)
				if len(fields) >= 2 {
					current.IPv6 = strings.Split(fields[1], "%")[0]
				}
			}
			if strings.HasPrefix(trimmed, "ether ") {
				fields := strings.Fields(trimmed)
				if len(fields) >= 2 {
					current.MAC = fields[1]
				}
			}
			if strings.Contains(trimmed, "status: inactive") {
				current.Status = "down"
			}
		}
	}
	if current != nil && current.Name != "lo0" {
		interfaces = append(interfaces, *current)
	}

	return interfaces
}

func guessInterfaceType(name string) string {
	switch {
	case name == "lo" || name == "lo0":
		return "loopback"
	case strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth"):
		return "ethernet"
	case strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "wlan"):
		return "wifi"
	case strings.HasPrefix(name, "br") || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "cni"):
		return "bridge"
	case strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "wg") || strings.HasPrefix(name, "tailscale"):
		return "tunnel"
	case strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "cali") || strings.HasPrefix(name, "flannel"):
		return "virtual"
	default:
		return "other"
	}
}
