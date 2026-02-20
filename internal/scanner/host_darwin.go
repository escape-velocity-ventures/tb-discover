package scanner

import (
	"context"
	"strconv"
	"strings"
)

func collectHostInfo(ctx context.Context, runner CommandRunner, info *HostInfo) error {
	// OS version from sw_vers
	if out, err := runner.Run(ctx, "sw_vers -productVersion"); err == nil {
		info.System.OSVersion = strings.TrimSpace(string(out))
	}

	// CPU model
	if out, err := runner.Run(ctx, "sysctl -n machdep.cpu.brand_string"); err == nil {
		info.System.CPUModel = strings.TrimSpace(string(out))
	}

	// CPU cores (physical)
	if out, err := runner.Run(ctx, "sysctl -n hw.physicalcpu"); err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			info.System.CPUCores = cores
		}
	}

	// Memory in GB
	if out, err := runner.Run(ctx, "sysctl -n hw.memsize"); err == nil {
		if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
			info.System.MemoryGB = float64(bytes) / (1024 * 1024 * 1024)
		}
	}

	return nil
}
