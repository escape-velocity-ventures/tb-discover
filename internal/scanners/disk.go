package scanners

import (
	"strings"
)

// ScanDisk parses df -h output for filesystem utilization.
func ScanDisk() []DiskInfo {
	result := HostExec("df -h")
	if result.ExitCode != 0 {
		return nil
	}
	return ParseDf(result.Stdout)
}

// ParseDf parses df -h output into DiskInfo entries.
// Exported for testing.
func ParseDf(output string) []DiskInfo {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil
	}

	var disks []DiskInfo
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		mount := fields[len(fields)-1]

		// Skip system mounts
		if strings.HasPrefix(mount, "/System") ||
			strings.HasPrefix(mount, "/dev") ||
			strings.HasPrefix(mount, "/proc") ||
			strings.HasPrefix(mount, "/sys") ||
			strings.HasPrefix(mount, "/run") ||
			strings.HasPrefix(mount, "/snap/") ||
			mount == "/private/var/vm" {
			continue
		}

		// Skip virtual/overlay filesystems
		fs := fields[0]
		if fs == "tmpfs" || fs == "devtmpfs" || fs == "overlay" || fs == "squashfs" {
			continue
		}

		disks = append(disks, DiskInfo{
			Filesystem: fs,
			Size:       fields[1],
			Used:       fields[2],
			Available:  fields[3],
			UsePercent: fields[4],
			Mount:      mount,
		})
	}

	return disks
}
