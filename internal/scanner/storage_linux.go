package scanner

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

func collectStorageInfo(ctx context.Context, runner CommandRunner, info *StorageInfo) error {
	// Filesystem info from df
	if out, err := runner.Run(ctx, "df -Pk 2>/dev/null"); err == nil {
		info.Filesystems = parseDfOutput(string(out))
	}

	// Disk info from lsblk (Linux only)
	if out, err := runner.Run(ctx, "lsblk -J -b -o NAME,SIZE,TYPE,MODEL,SERIAL,RO 2>/dev/null"); err == nil {
		info.Disks = parseLsblkJSON(out)
	}

	return nil
}

// parseDfOutput parses `df -k` output (sizes in 1K blocks).
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
		if !strings.HasPrefix(fs, "/") && !strings.Contains(fs, ":") {
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

		mountPoint := strings.Join(fields[5:], " ")

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

type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name     string         `json:"name"`
	Size     json.Number    `json:"size"`
	Type     string         `json:"type"`
	Model    *string        `json:"model"`
	Serial   *string        `json:"serial"`
	ReadOnly json.Number    `json:"ro"`
	Children []lsblkDevice  `json:"children,omitempty"`
}

func parseLsblkJSON(data []byte) []DiskInfo {
	var output lsblkOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil
	}

	var disks []DiskInfo
	for _, dev := range output.Blockdevices {
		disks = append(disks, lsblkDeviceToDisk(dev)...)
	}
	return disks
}

func lsblkDeviceToDisk(dev lsblkDevice) []DiskInfo {
	sizeBytes, _ := dev.Size.Int64()
	ro, _ := dev.ReadOnly.Int64()

	disk := DiskInfo{
		Name:     dev.Name,
		SizeGB:   float64(sizeBytes) / (1024 * 1024 * 1024),
		Type:     dev.Type,
		ReadOnly: ro != 0,
	}
	if dev.Model != nil {
		disk.Model = *dev.Model
	}
	if dev.Serial != nil {
		disk.Serial = *dev.Serial
	}

	result := []DiskInfo{disk}
	for _, child := range dev.Children {
		result = append(result, lsblkDeviceToDisk(child)...)
	}
	return result
}
