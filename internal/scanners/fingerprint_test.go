package scanners

import (
	"testing"
)

func TestComputeHardwareID_Deterministic(t *testing.T) {
	interfaces := []NetworkInterface{
		{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:01"},
		{Name: "eth1", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:02"},
	}
	storage := []StorageDevice{
		{Device: "/dev/sda", Serial: "WD-ABC123"},
	}

	id1 := ComputeHardwareID(interfaces, storage)
	id2 := ComputeHardwareID(interfaces, storage)

	if id1 != id2 {
		t.Errorf("hardware ID not deterministic: %s != %s", id1, id2)
	}
	if id1 == "" {
		t.Error("hardware ID should not be empty with valid inputs")
	}
	if len(id1) != 24 {
		t.Errorf("expected 24-char hex string, got %d chars: %s", len(id1), id1)
	}
}

func TestComputeHardwareID_OrderIndependent(t *testing.T) {
	ifaces1 := []NetworkInterface{
		{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:01"},
		{Name: "eth1", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:02"},
	}
	// Reverse order — same hardware, different enumeration order
	ifaces2 := []NetworkInterface{
		{Name: "eth1", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:02"},
		{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:01"},
	}
	storage := []StorageDevice{
		{Device: "/dev/sda", Serial: "WD-ABC123"},
	}

	id1 := ComputeHardwareID(ifaces1, storage)
	id2 := ComputeHardwareID(ifaces2, storage)

	if id1 != id2 {
		t.Errorf("hardware ID should be order-independent: %s != %s", id1, id2)
	}
}

func TestComputeHardwareID_SkipsVirtualInterfaces(t *testing.T) {
	physOnly := []NetworkInterface{
		{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:01"},
	}
	withVirtual := []NetworkInterface{
		{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:01"},
		{Name: "docker0", Type: "bridge", MAC: "02:42:ac:11:00:01"},
		{Name: "veth123", Type: "virtual", MAC: "fe:ed:be:ef:00:01"},
	}

	id1 := ComputeHardwareID(physOnly, nil)
	id2 := ComputeHardwareID(withVirtual, nil)

	if id1 != id2 {
		t.Errorf("virtual interfaces should not affect hardware ID: %s != %s", id1, id2)
	}
}

func TestComputeHardwareID_Empty(t *testing.T) {
	id := ComputeHardwareID(nil, nil)
	if id != "" {
		t.Errorf("expected empty hardware ID with no inputs, got %s", id)
	}

	id = ComputeHardwareID(
		[]NetworkInterface{{Name: "docker0", Type: "bridge", MAC: "02:42:ac:11:00:01"}},
		[]StorageDevice{{Device: "/dev/sda"}}, // no serial
	)
	if id != "" {
		t.Errorf("expected empty hardware ID with no physical MACs and no serials, got %s", id)
	}
}

func TestComputeHardwareID_CaseInsensitiveMAC(t *testing.T) {
	lower := []NetworkInterface{{Name: "eth0", Type: "ethernet", MAC: "aa:bb:cc:dd:ee:ff"}}
	upper := []NetworkInterface{{Name: "eth0", Type: "ethernet", MAC: "AA:BB:CC:DD:EE:FF"}}

	id1 := ComputeHardwareID(lower, nil)
	id2 := ComputeHardwareID(upper, nil)

	if id1 != id2 {
		t.Errorf("MAC comparison should be case-insensitive: %s != %s", id1, id2)
	}
}
