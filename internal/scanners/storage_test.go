package scanners

import (
	"testing"
)

func TestParseLsblkJSON_NVMe(t *testing.T) {
	jsonData := loadTestData(t, "lsblk-nvme.json")
	devices := ParseLsblkJSON(jsonData)

	// Should have 2 disks: nvme0n1 and sda (loop0 filtered)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	// NVMe drive
	nvme := devices[0]
	if nvme.Device != "/dev/nvme0n1" {
		t.Errorf("expected /dev/nvme0n1, got %s", nvme.Device)
	}
	if nvme.Model != "Samsung SSD 970 EVO Plus 500GB" {
		t.Errorf("expected Samsung SSD model, got %q", nvme.Model)
	}
	if nvme.Size != "465.8 GB" {
		t.Errorf("expected 465.8 GB, got %s", nvme.Size)
	}
	if nvme.Bus != "nvme" {
		t.Errorf("expected bus nvme, got %s", nvme.Bus)
	}
	if nvme.Protocol != "NVMe" {
		t.Errorf("expected protocol NVMe, got %s", nvme.Protocol)
	}
	if nvme.Removable {
		t.Error("NVMe should not be removable")
	}

	// NVMe partitions
	if len(nvme.Partitions) != 2 {
		t.Fatalf("expected 2 partitions on nvme, got %d", len(nvme.Partitions))
	}

	efi := nvme.Partitions[0]
	if efi.Device != "/dev/nvme0n1p1" {
		t.Errorf("expected /dev/nvme0n1p1, got %s", efi.Device)
	}
	if efi.FSType != "vfat" {
		t.Errorf("expected fstype vfat, got %s", efi.FSType)
	}
	if efi.MountPoint != "/boot/efi" {
		t.Errorf("expected mountpoint /boot/efi, got %s", efi.MountPoint)
	}
	if efi.Name != "EFI" {
		t.Errorf("expected label EFI, got %s", efi.Name)
	}

	root := nvme.Partitions[1]
	if root.FSType != "ext4" {
		t.Errorf("expected fstype ext4, got %s", root.FSType)
	}
	if root.MountPoint != "/" {
		t.Errorf("expected mountpoint /, got %s", root.MountPoint)
	}

	// SATA drive
	sata := devices[1]
	if sata.Device != "/dev/sda" {
		t.Errorf("expected /dev/sda, got %s", sata.Device)
	}
	if sata.Protocol != "SATA" {
		t.Errorf("expected protocol SATA, got %s", sata.Protocol)
	}
	if sata.Size != "1.8 TB" {
		t.Errorf("expected 1.8 TB, got %s", sata.Size)
	}

	// SATA partition
	if len(sata.Partitions) != 1 {
		t.Fatalf("expected 1 partition on sda, got %d", len(sata.Partitions))
	}
	dataPart := sata.Partitions[0]
	if dataPart.Name != "data" {
		t.Errorf("expected label data, got %s", dataPart.Name)
	}
	if dataPart.MountPoint != "/mnt/data" {
		t.Errorf("expected mountpoint /mnt/data, got %s", dataPart.MountPoint)
	}
}

func TestParseLsblkJSON_FiltersLoopDevices(t *testing.T) {
	jsonData := loadTestData(t, "lsblk-nvme.json")
	devices := ParseLsblkJSON(jsonData)

	for _, d := range devices {
		if d.Device == "/dev/loop0" {
			t.Error("loop device should be filtered")
		}
	}
}

func TestParseLsblkJSON_InvalidJSON(t *testing.T) {
	devices := ParseLsblkJSON("not json")
	if devices != nil {
		t.Errorf("expected nil for invalid JSON, got %+v", devices)
	}
}

func TestParseLsblkJSON_EmptyDevices(t *testing.T) {
	devices := ParseLsblkJSON(`{"blockdevices": []}`)
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"500107862016", "465.8 GB"},
		{"2000398934016", "1.8 TB"},
		{"536870912", "512 MB"},
		{"58720256", "56 MB"},
		{"0", "0"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tc := range tests {
		result := formatBytes(tc.input)
		if result != tc.expected {
			t.Errorf("formatBytes(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormalizeProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nvme", "NVMe"},
		{"sata", "SATA"},
		{"usb", "USB"},
		{"sas", "SAS"},
		{"ata", "ata"},
	}

	for _, tc := range tests {
		result := normalizeProtocol(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeProtocol(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
