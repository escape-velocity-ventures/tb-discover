package scanner

import (
	"context"
	"strings"

	"github.com/tinkerbelle-io/tb-discover/internal/scanner/parser"
)

func collectNetworkInfo(ctx context.Context, runner CommandRunner, info *NetworkInfo) error {
	// Parse ifconfig output
	if out, err := runner.Run(ctx, "ifconfig -a"); err == nil {
		info.Interfaces = convertParserInterfaces(parser.ParseIfconfig(string(out)))
	}

	// Parse routes
	if out, err := runner.Run(ctx, "netstat -rn -f inet"); err == nil {
		info.Routes = parseNetstatRoutes(string(out))
	}

	return nil
}

// parseNetstatRoutes parses macOS `netstat -rn` output.
func parseNetstatRoutes(output string) []RouteInfo {
	var routes []RouteInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Skip header lines
		if fields[0] == "Destination" || fields[0] == "Routing" || fields[0] == "Internet:" {
			continue
		}
		// Skip non-route lines
		if !strings.Contains(fields[0], ".") && fields[0] != "default" {
			continue
		}

		route := RouteInfo{
			Destination: fields[0],
			Gateway:     fields[1],
		}
		// Interface is typically the last field or the 4th
		if len(fields) >= 4 {
			// On macOS, interface is typically the last or 4th field
			for i := len(fields) - 1; i >= 3; i-- {
				if isInterfaceName(fields[i]) {
					route.Interface = fields[i]
					break
				}
			}
		}
		routes = append(routes, route)
	}

	return routes
}

func isInterfaceName(s string) bool {
	prefixes := []string{"en", "lo", "utun", "bridge", "awdl", "llw", "gif", "stf", "anpi", "ap", "vmnet"}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
