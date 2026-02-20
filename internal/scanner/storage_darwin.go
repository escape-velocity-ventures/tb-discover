package scanner

import (
	"context"
	"strconv"
	"strings"
)

func collectStorageInfo(ctx context.Context, runner CommandRunner, info *StorageInfo) error {
	// Use df -Pk for POSIX output (6 columns, consistent across platforms)
	if out, err := runner.Run(ctx, "df -Pk 2>/dev/null"); err == nil {
		info.Filesystems = parseDfOutput(string(out))
	}

	return nil
}

// parseDfOutput parses `df -Pk` output (POSIX format, sizes in 1K blocks).
// Columns: Filesystem 1024-blocks Used Available Capacity Mounted-on
func parseDfOutput(output string) []FilesystemInfo {
	var filesystems []FilesystemInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		// Skip header
		if fields[0] == "Filesystem" {
			continue
		}
		// Skip pseudo-filesystems
		fs := fields[0]
		if fs == "devfs" || fs == "map" || strings.HasPrefix(fs, "map ") {
			continue
		}

		sizeKB, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		usedKB, _ := strconv.ParseInt(fields[2], 10, 64)
		availKB, _ := strconv.ParseInt(fields[3], 10, 64)

		pctStr := strings.TrimSuffix(fields[4], "%")
		pct, _ := strconv.ParseFloat(pctStr, 64)

		// Mount point may contain spaces, join remaining fields
		mountPoint := strings.Join(fields[5:], " ")

		// Skip if mount point doesn't start with /
		if !strings.HasPrefix(mountPoint, "/") {
			continue
		}

		filesystems = append(filesystems, FilesystemInfo{
			Filesystem: fs,
			MountPoint: mountPoint,
			SizeGB:     float64(sizeKB) / (1024 * 1024),
			UsedGB:     float64(usedKB) / (1024 * 1024),
			AvailGB:    float64(availKB) / (1024 * 1024),
			UsePct:     pct,
		})
	}

	return filesystems
}
