package scanner

import "github.com/tinkerbelle-io/tb-discover/internal/scanner/parser"

// convertParserInterfaces converts parser.InterfaceInfo to scanner.InterfaceInfo.
func convertParserInterfaces(in []parser.InterfaceInfo) []InterfaceInfo {
	out := make([]InterfaceInfo, len(in))
	for i, iface := range in {
		out[i] = InterfaceInfo{
			Name:  iface.Name,
			IP:    iface.IP,
			IPv6:  iface.IPv6,
			MAC:   iface.MAC,
			MTU:   iface.MTU,
			State: iface.State,
			Type:  iface.Type,
		}
	}
	return out
}
