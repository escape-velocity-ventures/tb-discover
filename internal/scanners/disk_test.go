package scanners

import (
	"os"
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

	// EFI partition
	efi := disks[1]
	if efi.Mount != "/boot/efi" {
		t.Errorf("expected mount /boot/efi, got %s", efi.Mount)
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
