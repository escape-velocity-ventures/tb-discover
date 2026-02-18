package scanners

import (
	"regexp"
	"strconv"
	"strings"
)

// ScanResources gathers load average, memory pressure, and uptime.
func ScanResources() *ResourceInfo {
	info := &ResourceInfo{}

	// Load average
	if content, err := ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(content)
		if len(fields) >= 3 {
			for i := 0; i < 3; i++ {
				if v, err := strconv.ParseFloat(fields[i], 64); err == nil {
					info.LoadAvg[i] = v
				}
			}
		}
	}

	// Memory usage
	if content, err := ReadFile("/proc/meminfo"); err == nil {
		mem := parseMeminfo(content)
		if mem != nil {
			info.Memory = mem
		}
	}

	// Uptime
	result := HostExec("uptime -s 2>/dev/null || uptime")
	if result.ExitCode == 0 {
		info.Uptime = strings.TrimSpace(result.Stdout)
	}

	return info
}

func parseMeminfo(content string) *MemoryInfo {
	values := make(map[string]int64)
	re := regexp.MustCompile(`^(\w+):\s+(\d+)\s+kB`)

	for _, line := range strings.Split(content, "\n") {
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			if kb, err := strconv.ParseInt(m[2], 10, 64); err == nil {
				values[m[1]] = kb
			}
		}
	}

	total, hasTotal := values["MemTotal"]
	available, hasAvail := values["MemAvailable"]

	if !hasTotal {
		return nil
	}

	if !hasAvail {
		// Estimate available from free + buffers + cached
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}

	used := total - available
	totalGB := float64(total) / 1024 / 1024
	usedGB := float64(used) / 1024 / 1024
	availGB := float64(available) / 1024 / 1024

	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}

	return &MemoryInfo{
		TotalGB:     round2(totalGB),
		UsedGB:      round2(usedGB),
		AvailableGB: round2(availGB),
		UsePercent:  round2(pct),
	}
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}
