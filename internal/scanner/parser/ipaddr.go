package parser

import (
	"strconv"
	"strings"
)

// IPAddrJSON represents the JSON output of `ip -j addr show`.
type IPAddrJSON struct {
	IfName    string         `json:"ifname"`
	Mtu       int            `json:"mtu"`
	Operstate string         `json:"operstate"`
	Address   string         `json:"address"` // MAC
	AddrInfo  []IPAddrInfo   `json:"addr_info"`
}

// IPAddrInfo is an address entry within ip -j output.
type IPAddrInfo struct {
	Family    string `json:"family"` // "inet" or "inet6"
	Local     string `json:"local"`
	Prefixlen int    `json:"prefixlen"`
	Scope     string `json:"scope"`
}

// IPRouteJSON represents the JSON output of `ip -j route show`.
type IPRouteJSON struct {
	Dst     string `json:"dst"`
	Gateway string `json:"gateway"`
	Dev     string `json:"dev"`
	Metric  int    `json:"metric"`
}

// ParseIPAddrJSON converts parsed JSON to InterfaceInfo slice.
func ParseIPAddrJSON(addrs []IPAddrJSON) []InterfaceInfo {
	var interfaces []InterfaceInfo

	for _, a := range addrs {
		iface := InterfaceInfo{
			Name:  a.IfName,
			MAC:   a.Address,
			MTU:   a.Mtu,
			State: strings.ToLower(a.Operstate),
		}

		for _, addr := range a.AddrInfo {
			switch addr.Family {
			case "inet":
				if iface.IP == "" {
					iface.IP = addr.Local
				}
			case "inet6":
				if iface.IPv6 == "" && addr.Scope != "link" {
					iface.IPv6 = addr.Local
				}
			}
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces
}

// ParseIPAddr parses text output of `ip addr show`.
func ParseIPAddr(output string) []InterfaceInfo {
	var interfaces []InterfaceInfo
	var current *InterfaceInfo

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		// Interface line: "2: eth0: <BROADCAST,...> mtu 1500 ... state UP ..."
		if len(line) > 0 && line[0] != ' ' {
			if current != nil {
				interfaces = append(interfaces, *current)
			}
			current = &InterfaceInfo{}

			fields := strings.Fields(line)
			if len(fields) >= 2 {
				current.Name = strings.TrimSuffix(fields[1], ":")
				// Remove @... suffix (e.g., "eth0@if5")
				if idx := strings.Index(current.Name, "@"); idx >= 0 {
					current.Name = current.Name[:idx]
				}
			}

			for i := 0; i < len(fields)-1; i++ {
				switch fields[i] {
				case "mtu":
					if mtu, err := strconv.Atoi(fields[i+1]); err == nil {
						current.MTU = mtu
					}
				case "state":
					current.State = strings.ToLower(fields[i+1])
				}
			}

			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// "link/ether aa:bb:cc:dd:ee:ff brd ff:ff:ff:ff:ff:ff"
		if strings.HasPrefix(trimmed, "link/ether ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				current.MAC = fields[1]
			}
		}

		// "inet 10.0.0.1/24 ..."
		if strings.HasPrefix(trimmed, "inet ") && current.IP == "" {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				ip := fields[1]
				// Strip CIDR prefix
				if idx := strings.Index(ip, "/"); idx >= 0 {
					ip = ip[:idx]
				}
				current.IP = ip
			}
		}

		// "inet6 2001:db8::1/64 scope global"
		if strings.HasPrefix(trimmed, "inet6 ") && current.IPv6 == "" {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				addr := fields[1]
				if idx := strings.Index(addr, "/"); idx >= 0 {
					addr = addr[:idx]
				}
				// Skip link-local
				if !strings.HasPrefix(addr, "fe80") {
					// Check scope
					scopeOK := true
					for i, f := range fields {
						if f == "scope" && i+1 < len(fields) && fields[i+1] == "link" {
							scopeOK = false
						}
					}
					if scopeOK {
						current.IPv6 = addr
					}
				}
			}
		}
	}

	if current != nil {
		interfaces = append(interfaces, *current)
	}

	return interfaces
}
