package scanners

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// ScanStorage discovers physical/virtual block devices.
func ScanStorage() []StorageDevice {
	switch runtime.GOOS {
	case "linux":
		return scanLinuxStorage()
	case "darwin":
		return scanDarwinStorage()
	default:
		return nil
	}
}

// lsblkOutput matches the JSON structure from lsblk -J.
type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       string        `json:"size"`
	Type       string        `json:"type"`       // "disk", "part", "rom", "loop"
	MountPoint *string       `json:"mountpoint"` // nullable
	FSType     *string       `json:"fstype"`     // nullable
	Model      *string       `json:"model"`      // nullable
	Serial     *string       `json:"serial"`     // nullable — hardware serial number
	RM         json.Number   `json:"rm"`         // removable: "0" or "1" or bool
	Tran       *string       `json:"tran"`        // transport: "sata", "nvme", "usb"
	Label      *string       `json:"label"`       // nullable
	Children   []lsblkDevice `json:"children"`
}

func scanLinuxStorage() []StorageDevice {
	result := HostExec("lsblk -J -b -o NAME,SIZE,TYPE,MOUNTPOINT,FSTYPE,MODEL,SERIAL,RM,TRAN,LABEL 2>/dev/null")
	if result.ExitCode != 0 {
		return scanLinuxStorageFallback()
	}

	devices := ParseLsblkJSON(result.Stdout)
	if devices == nil {
		return scanLinuxStorageFallback()
	}
	return devices
}

// ParseLsblkJSON parses lsblk -J -b output into StorageDevice entries.
// Exported for testing.
func ParseLsblkJSON(jsonData string) []StorageDevice {
	var parsed lsblkOutput
	if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
		return nil
	}

	var devices []StorageDevice
	for _, bd := range parsed.Blockdevices {
		// Only include actual disks (skip loop, rom)
		if bd.Type != "disk" {
			continue
		}

		dev := StorageDevice{
			Device:    "/dev/" + bd.Name,
			Size:      formatBytes(bd.Size),
			Removable: isRemovable(bd.RM),
		}

		if bd.Model != nil {
			dev.Model = strings.TrimSpace(*bd.Model)
		}
		if bd.Serial != nil {
			dev.Serial = strings.TrimSpace(*bd.Serial)
		}
		if bd.Tran != nil {
			dev.Bus = *bd.Tran
			dev.Protocol = normalizeProtocol(*bd.Tran)
		}

		// Partitions
		for _, child := range bd.Children {
			part := StoragePartition{
				Device: "/dev/" + child.Name,
				Size:   formatBytes(child.Size),
			}
			if child.FSType != nil {
				part.FSType = *child.FSType
			}
			if child.MountPoint != nil {
				part.MountPoint = *child.MountPoint
			}
			if child.Label != nil {
				part.Name = *child.Label
			}
			dev.Partitions = append(dev.Partitions, part)
		}

		if dev.Partitions == nil {
			dev.Partitions = []StoragePartition{}
		}

		devices = append(devices, dev)
	}

	return devices
}

func scanLinuxStorageFallback() []StorageDevice {
	// Simple fallback using lsblk without JSON
	result := HostExec("lsblk -d -o NAME,SIZE,TYPE,MODEL 2>/dev/null")
	if result.ExitCode != 0 {
		return nil
	}

	var devices []StorageDevice
	for _, line := range strings.Split(result.Stdout, "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[2] != "disk" {
			continue
		}

		dev := StorageDevice{
			Device:     "/dev/" + fields[0],
			Size:       fields[1],
			Removable:  false,
			Partitions: []StoragePartition{},
		}
		if len(fields) >= 4 {
			dev.Model = strings.Join(fields[3:], " ")
		}
		devices = append(devices, dev)
	}

	return devices
}

func scanDarwinStorage() []StorageDevice {
	// macOS: use diskutil list
	result := HostExec("diskutil list -plist 2>/dev/null")
	if result.ExitCode != 0 {
		return nil
	}
	// For now, fall back to simple diskutil output
	result = HostExec("diskutil list 2>/dev/null")
	if result.ExitCode != 0 {
		return nil
	}

	// Basic parsing of diskutil list output
	var devices []StorageDevice
	var current *StorageDevice

	for _, line := range strings.Split(result.Stdout, "\n") {
		if strings.HasPrefix(line, "/dev/disk") {
			if current != nil {
				if current.Partitions == nil {
					current.Partitions = []StoragePartition{}
				}
				devices = append(devices, *current)
			}

			// Extract size from parenthetical
			parts := strings.SplitN(line, "(", 2)
			size := ""
			if len(parts) == 2 {
				// e.g., "(internal, physical):  500.1 GB"
				sizeField := strings.SplitN(parts[1], ")", 2)
				if len(sizeField) == 2 {
					size = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sizeField[1]), ":"))
				}
			}

			devName := strings.Fields(line)[0]
			current = &StorageDevice{
				Device:     devName,
				Size:       size,
				Removable:  false,
				Partitions: []StoragePartition{},
			}

			if strings.Contains(line, "internal") {
				current.Bus = "internal"
			}
			if strings.Contains(line, "external") {
				current.Removable = true
				current.Bus = "usb"
			}
		}
	}
	if current != nil {
		if current.Partitions == nil {
			current.Partitions = []StoragePartition{}
		}
		devices = append(devices, *current)
	}

	return devices
}

func isRemovable(rm json.Number) bool {
	s := rm.String()
	return s == "1" || s == "true"
}

func normalizeProtocol(tran string) string {
	switch strings.ToLower(tran) {
	case "nvme":
		return "NVMe"
	case "sata":
		return "SATA"
	case "usb":
		return "USB"
	case "sas":
		return "SAS"
	default:
		return tran
	}
}

func formatBytes(sizeStr string) string {
	bytes, err := strconv.ParseInt(strings.TrimSpace(sizeStr), 10, 64)
	if err != nil || bytes == 0 {
		return sizeStr
	}

	const (
		MB = 1024 * 1024
		GB = 1024 * 1024 * 1024
		TB = 1024 * 1024 * 1024 * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%d MB", bytes/MB)
	default:
		return sizeStr
	}
}
