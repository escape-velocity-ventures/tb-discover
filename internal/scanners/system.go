package scanners

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// ScanSystem gathers OS, CPU, memory, and architecture info.
func ScanSystem() SystemInfo {
	info := SystemInfo{
		OS:   runtime.GOOS,
		Arch: normalizeArch(runtime.GOARCH),
	}

	switch runtime.GOOS {
	case "linux":
		scanLinuxSystem(&info)
	case "darwin":
		scanDarwinSystem(&info)
	}

	return info
}

func scanLinuxSystem(info *SystemInfo) {
	// OS version from /etc/os-release
	if content, err := ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.OSVersion = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	if info.OSVersion == "" {
		info.OSVersion = "Linux"
	}

	// CPU info from /proc/cpuinfo
	if content, err := ReadFile("/proc/cpuinfo"); err == nil {
		cores := 0
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "processor") {
				cores++
			}
			if strings.HasPrefix(line, "model name") && info.CPUModel == "" {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPUModel = strings.TrimSpace(parts[1])
				}
			}
		}
		info.CPUCores = cores
	}

	// Memory from /proc/meminfo
	if content, err := ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				re := regexp.MustCompile(`(\d+)`)
				if m := re.FindString(line); m != "" {
					if kb, err := strconv.ParseInt(m, 10, 64); err == nil {
						info.MemoryGB = int(kb / 1024 / 1024)
					}
				}
			}
		}
	}

	// Hostname
	result := HostExec("hostname -f 2>/dev/null || hostname")
	if result.ExitCode == 0 {
		// Hostname is captured in the Network section, but we also use it for OS version context
		_ = strings.TrimSpace(result.Stdout)
	}
}

func scanDarwinSystem(info *SystemInfo) {
	if result := HostExec("sw_vers -productVersion"); result.ExitCode == 0 {
		info.OSVersion = "macOS " + strings.TrimSpace(result.Stdout)
	}

	if result := HostExec("sysctl -n machdep.cpu.brand_string"); result.ExitCode == 0 {
		info.CPUModel = strings.TrimSpace(result.Stdout)
	}

	if result := HostExec("sysctl -n hw.ncpu"); result.ExitCode == 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(result.Stdout)); err == nil {
			info.CPUCores = n
		}
	}

	if result := HostExec("sysctl -n hw.memsize"); result.ExitCode == 0 {
		if bytes, err := strconv.ParseInt(strings.TrimSpace(result.Stdout), 10, 64); err == nil {
			info.MemoryGB = int(bytes / 1024 / 1024 / 1024)
		}
	}
}

// GetHostname returns the machine hostname.
func GetHostname() string {
	result := HostExec("hostname -f 2>/dev/null || hostname")
	if result.ExitCode == 0 {
		return strings.TrimSpace(result.Stdout)
	}
	return "unknown"
}

func normalizeArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return goarch
	}
}
