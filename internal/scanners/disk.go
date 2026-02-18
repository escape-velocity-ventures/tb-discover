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
// Handles both Linux (6 cols) and macOS (9 cols) df formats,
// including mount points that contain spaces.
// Exported for testing.
func ParseDf(output string) []DiskInfo {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil
	}

	// Detect macOS format by checking for "Capacity" or "iused" in header
	header := lines[0]
	isMacOS := strings.Contains(header, "Capacity") || strings.Contains(header, "iused")

	// Find the column position of "Mounted on" in the header
	mountCol := strings.Index(header, "Mounted on")

	var disks []DiskInfo
	for _, line := range lines[1:] { // skip header
		if len(line) == 0 {
			continue
		}

		fields := strings.Fields(line)
		minFields := 6
		if isMacOS {
			minFields = 9
		}
		if len(fields) < minFields {
			continue
		}

		// Extract mount point — use column position if available, otherwise last field
		var mount string
		if mountCol > 0 && len(line) > mountCol {
			mount = strings.TrimSpace(line[mountCol:])
		} else {
			mount = fields[len(fields)-1]
		}

		fs := fields[0]

		// Extract size/used/avail/use% at correct column positions
		var size, used, avail, usePct string
		if isMacOS {
			size = fields[1]
			used = fields[2]
			avail = fields[3]
			usePct = fields[4] // "Capacity" column (e.g., "7%")
		} else {
			size = fields[1]
			used = fields[2]
			avail = fields[3]
			usePct = fields[4]
		}

		if shouldSkipMount(mount, fs) {
			continue
		}

		disks = append(disks, DiskInfo{
			Filesystem: fs,
			Size:       size,
			Used:       used,
			Available:  avail,
			UsePercent: usePct,
			Mount:      mount,
			Origin:     classifyDiskOrigin(fs),
		})
	}

	return disks
}

// shouldSkipMount returns true for mounts and filesystems we should exclude.
func shouldSkipMount(mount, fs string) bool {
	// Skip virtual/overlay/system filesystems
	switch fs {
	case "tmpfs", "devtmpfs", "overlay", "squashfs", "devfs", "none":
		return true
	}
	if fs == "map" || strings.HasPrefix(fs, "map ") {
		return true
	}

	// macOS Time Machine snapshots (fs starts with "com.apple.TimeMachine.")
	if strings.HasPrefix(fs, "com.apple.") {
		return true
	}

	// Skip system mounts
	if strings.HasPrefix(mount, "/System") ||
		strings.HasPrefix(mount, "/dev") ||
		strings.HasPrefix(mount, "/proc") ||
		strings.HasPrefix(mount, "/sys") ||
		strings.HasPrefix(mount, "/run") ||
		strings.HasPrefix(mount, "/snap/") ||
		mount == "/private/var/vm" {
		return true
	}

	// Skip macOS noise: Time Machine, CoreSimulator, Xcode, FUSE/app mounts
	if strings.Contains(mount, ".timemachine") ||
		strings.Contains(mount, "CoreSimulator") ||
		strings.Contains(mount, "/Developer/") ||
		strings.HasPrefix(mount, "/Volumes/com.apple.") ||
		strings.HasPrefix(mount, "/private/var/folders/") {
		return true
	}

	return false
}

// classifyDiskOrigin determines if a filesystem is local, network, or virtual.
func classifyDiskOrigin(filesystem string) string {
	// Network: NFS, CIFS/SMB, GlusterFS, CephFS — device string contains ":"  or "//"
	if strings.Contains(filesystem, ":") || strings.HasPrefix(filesystem, "//") {
		return "network"
	}

	// Known network filesystem types
	switch filesystem {
	case "nfs", "nfs4", "cifs", "smb", "glusterfs", "ceph", "fuse.sshfs":
		return "network"
	}

	// Local block devices
	if strings.HasPrefix(filesystem, "/dev/") {
		return "local"
	}

	// macOS local volumes
	if strings.HasPrefix(filesystem, "/dev") || strings.HasPrefix(filesystem, "disk") {
		return "local"
	}

	return "virtual"
}
