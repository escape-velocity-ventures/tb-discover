package scanners

import (
	"math"
	"testing"
)

func TestParseMeminfo(t *testing.T) {
	content := loadTestData(t, "meminfo.txt")
	mem := parseMeminfo(content)

	if mem == nil {
		t.Fatal("expected non-nil MemoryInfo")
	}

	// MemTotal: 16384000 kB = 15.625 GB
	expectedTotal := 15.62
	if math.Abs(mem.TotalGB-expectedTotal) > 0.01 {
		t.Errorf("expected TotalGB ~%.2f, got %.2f", expectedTotal, mem.TotalGB)
	}

	// MemAvailable: 8192000 kB = 7.8125 GB
	expectedAvail := 7.81
	if math.Abs(mem.AvailableGB-expectedAvail) > 0.01 {
		t.Errorf("expected AvailableGB ~%.2f, got %.2f", expectedAvail, mem.AvailableGB)
	}

	// Used = Total - Available = 16384000 - 8192000 = 8192000 kB = 7.8125 GB
	expectedUsed := 7.81
	if math.Abs(mem.UsedGB-expectedUsed) > 0.01 {
		t.Errorf("expected UsedGB ~%.2f, got %.2f", expectedUsed, mem.UsedGB)
	}

	// UsePercent = 8192000/16384000 * 100 = 50%
	if math.Abs(mem.UsePercent-50.0) > 0.1 {
		t.Errorf("expected UsePercent ~50%%, got %.2f%%", mem.UsePercent)
	}
}

func TestParseMeminfo_NoAvailable(t *testing.T) {
	content := loadTestData(t, "meminfo-no-available.txt")
	mem := parseMeminfo(content)

	if mem == nil {
		t.Fatal("expected non-nil MemoryInfo even without MemAvailable")
	}

	// Should estimate available = Free + Buffers + Cached
	// 2048000 + 512000 + 5632000 = 8192000 kB
	expectedAvail := 7.81
	if math.Abs(mem.AvailableGB-expectedAvail) > 0.01 {
		t.Errorf("expected estimated AvailableGB ~%.2f, got %.2f", expectedAvail, mem.AvailableGB)
	}
}

func TestParseMeminfo_Empty(t *testing.T) {
	mem := parseMeminfo("")
	if mem != nil {
		t.Errorf("expected nil for empty meminfo, got %+v", mem)
	}
}

func TestParseMeminfo_NoTotal(t *testing.T) {
	mem := parseMeminfo("MemFree:         2048000 kB\nMemAvailable:    8192000 kB\n")
	if mem != nil {
		t.Errorf("expected nil without MemTotal, got %+v", mem)
	}
}

func TestRound2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{15.625, 15.62},
		{7.8125, 7.81},
		{0.0, 0.0},
		{99.999, 99.99},
		{1.005, 1.0},
	}

	for _, tc := range tests {
		result := round2(tc.input)
		if result != tc.expected {
			t.Errorf("round2(%f) = %f, expected %f", tc.input, result, tc.expected)
		}
	}
}
