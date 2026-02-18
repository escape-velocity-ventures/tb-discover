package scanners

import (
	"os"
	"strings"
	"testing"
)

func loadTestData(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../test/testdata/" + name)
	if err != nil {
		t.Fatalf("failed to load test data %s: %v", name, err)
	}
	return string(data)
}

func TestParseDf_Ubuntu(t *testing.T) {
	output := loadTestData(t, "df-ubuntu.txt")
	disks := ParseDf(output)

	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}

	// Root filesystem
	root := disks[0]
	if root.Mount != "/" {
		t.Errorf("expected mount /, got %s", root.Mount)
	}
	if root.Filesystem != "/dev/sda2" {
		t.Errorf("expected filesystem /dev/sda2, got %s", root.Filesystem)
	}
	if root.Size != "457G" {
		t.Errorf("expected size 457G, got %s", root.Size)
	}
	if root.UsePercent != "5%" {
		t.Errorf("expected use 5%%, got %s", root.UsePercent)
	}
	if root.Origin != "local" {
		t.Errorf("expected origin local, got %s", root.Origin)
	}

	// EFI partition
	efi := disks[1]
	if efi.Mount != "/boot/efi" {
		t.Errorf("expected mount /boot/efi, got %s", efi.Mount)
	}
	if efi.Origin != "local" {
		t.Errorf("expected origin local, got %s", efi.Origin)
	}
}

func TestParseDf_FiltersSnap(t *testing.T) {
	output := loadTestData(t, "df-snap.txt")
	disks := ParseDf(output)

	// Should only include real filesystems, not snap/loop/squashfs
	for _, d := range disks {
		if d.Filesystem == "squashfs" {
			t.Errorf("squashfs should be filtered: %s", d.Mount)
		}
		if len(d.Mount) > 5 && d.Mount[:6] == "/snap/" {
			t.Errorf("/snap/ mounts should be filtered: %s", d.Mount)
		}
	}

	// Should have root + efi
	if len(disks) != 2 {
		t.Fatalf("expected 2 disks after snap filtering, got %d: %+v", len(disks), disks)
	}
}

func TestParseDf_FiltersSystemMounts(t *testing.T) {
	output := "Filesystem      Size  Used Avail Use% Mounted on\n" +
		"udev            7.8G     0  7.8G   0% /dev\n" +
		"tmpfs           1.6G  2.1M  1.6G   1% /run\n" +
		"/dev/sda1       457G   22G  412G   5% /\n" +
		"overlay          50G   10G   40G  20% /var/lib/docker\n" +
		"devtmpfs        7.8G     0  7.8G   0% /dev/pts\n"

	disks := ParseDf(output)

	if len(disks) != 1 {
		t.Fatalf("expected 1 disk after filtering, got %d: %+v", len(disks), disks)
	}
	if disks[0].Mount != "/" {
		t.Errorf("expected /, got %s", disks[0].Mount)
	}
}

func TestParseDf_Empty(t *testing.T) {
	disks := ParseDf("")
	if disks != nil {
		t.Errorf("expected nil for empty input, got %+v", disks)
	}

	disks = ParseDf("Filesystem      Size  Used Avail Use% Mounted on\n")
	if len(disks) != 0 {
		t.Errorf("expected 0 disks for header-only input, got %d", len(disks))
	}
}

func TestParseDf_NetworkOrigin(t *testing.T) {
	output := "Filesystem      Size  Used Avail Use% Mounted on\n" +
		"nas:/vol/share   10T  5.0T  5.0T  50% /mnt/nas\n" +
		"//server/share   2.0T  1.0T  1.0T  50% /mnt/smb\n" +
		"/dev/sda1       457G   22G  412G   5% /\n"

	disks := ParseDf(output)

	if len(disks) != 3 {
		t.Fatalf("expected 3 disks, got %d", len(disks))
	}

	if disks[0].Origin != "network" {
		t.Errorf("NFS mount should have origin network, got %s", disks[0].Origin)
	}
	if disks[1].Origin != "network" {
		t.Errorf("SMB mount should have origin network, got %s", disks[1].Origin)
	}
	if disks[2].Origin != "local" {
		t.Errorf("/dev/ mount should have origin local, got %s", disks[2].Origin)
	}
}

func TestParseDf_MacOS(t *testing.T) {
	output := loadTestData(t, "df-macos.txt")
	disks := ParseDf(output)

	// Expected: /, /Volumes/Media, /Volumes/Untitled, /Volumes/TESLADRIVE,
	// /Volumes/Vox 0.2.0, /Volumes/Backups of plato 1
	// Filtered: devfs, /System/*, map auto_home, CoreSimulator (x4),
	//           CloudEdge (/private/var/folders), Time Machine (x3),
	//           com.apple.TimeMachine.* (x2)
	expectedMounts := []string{
		"/",
		"/Volumes/Media",
		"/Volumes/Untitled",
		"/Volumes/TESLADRIVE",
		"/Volumes/Vox 0.2.0",
		"/Volumes/Backups of plato 1",
	}

	if len(disks) != len(expectedMounts) {
		t.Fatalf("expected %d disks, got %d:", len(expectedMounts), len(disks))
		for _, d := range disks {
			t.Logf("  %s → %s", d.Filesystem, d.Mount)
		}
	}

	for i, expected := range expectedMounts {
		if disks[i].Mount != expected {
			t.Errorf("disk[%d]: expected mount %q, got %q", i, expected, disks[i].Mount)
		}
	}

	// Verify mount points with spaces are parsed correctly
	vox := findDisk(disks, "/Volumes/Vox 0.2.0")
	if vox == nil {
		t.Fatal("mount with spaces '/Volumes/Vox 0.2.0' not found")
	}
	if vox.Filesystem != "/dev/disk19s1" {
		t.Errorf("Vox filesystem: expected /dev/disk19s1, got %s", vox.Filesystem)
	}

	backups := findDisk(disks, "/Volumes/Backups of plato 1")
	if backups == nil {
		t.Fatal("mount with spaces '/Volumes/Backups of plato 1' not found")
	}
	if backups.Size != "15Ti" {
		t.Errorf("Backups size: expected 15Ti, got %s", backups.Size)
	}

	// Time Machine network share should be filtered
	for _, d := range disks {
		if strings.Contains(d.Mount, ".timemachine") {
			t.Errorf("Time Machine mount should be filtered: %s", d.Mount)
		}
	}
}

func findDisk(disks []DiskInfo, mount string) *DiskInfo {
	for i := range disks {
		if disks[i].Mount == mount {
			return &disks[i]
		}
	}
	return nil
}

func TestClassifyDiskOrigin(t *testing.T) {
	tests := []struct {
		filesystem string
		expected   string
	}{
		{"/dev/sda1", "local"},
		{"/dev/nvme0n1p1", "local"},
		{"nas:/vol/share", "network"},
		{"//server/share", "network"},
		{"nfs", "network"},
		{"nfs4", "network"},
		{"cifs", "network"},
		{"ceph", "network"},
		{"fuse.sshfs", "network"},
		{"192.168.1.100:/data", "network"},
		{"fuse.lxcfs", "virtual"},
	}

	for _, tc := range tests {
		result := classifyDiskOrigin(tc.filesystem)
		if result != tc.expected {
			t.Errorf("classifyDiskOrigin(%q) = %q, expected %q", tc.filesystem, result, tc.expected)
		}
	}
}
