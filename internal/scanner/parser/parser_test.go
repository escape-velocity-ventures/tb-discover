package parser

import (
	"os"
	"testing"
)

func TestParseIfconfig(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/ifconfig_macos.txt")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	interfaces := ParseIfconfig(string(data))

	if len(interfaces) == 0 {
		t.Fatal("expected interfaces, got none")
	}

	// Build a map for easier lookup
	byName := make(map[string]InterfaceInfo)
	for _, iface := range interfaces {
		byName[iface.Name] = iface
	}

	// lo0
	if lo, ok := byName["lo0"]; ok {
		if lo.IP != "127.0.0.1" {
			t.Errorf("lo0 IP = %q, want 127.0.0.1", lo.IP)
		}
		if lo.MTU != 16384 {
			t.Errorf("lo0 MTU = %d, want 16384", lo.MTU)
		}
		if lo.State != "up" {
			t.Errorf("lo0 State = %q, want up", lo.State)
		}
	} else {
		t.Error("lo0 not found")
	}

	// en0
	if en0, ok := byName["en0"]; ok {
		if en0.IP != "192.168.1.100" {
			t.Errorf("en0 IP = %q, want 192.168.1.100", en0.IP)
		}
		if en0.MAC != "14:98:77:aa:bb:cc" {
			t.Errorf("en0 MAC = %q, want 14:98:77:aa:bb:cc", en0.MAC)
		}
		if en0.MTU != 1500 {
			t.Errorf("en0 MTU = %d, want 1500", en0.MTU)
		}
		if en0.IPv6 != "2600:1700:abc:def::1" {
			t.Errorf("en0 IPv6 = %q, want 2600:1700:abc:def::1", en0.IPv6)
		}
		if en0.State != "up" {
			t.Errorf("en0 State = %q, want up", en0.State)
		}
	} else {
		t.Error("en0 not found")
	}

	// en1 (no IP)
	if en1, ok := byName["en1"]; ok {
		if en1.IP != "" {
			t.Errorf("en1 IP = %q, want empty", en1.IP)
		}
		if en1.MAC != "36:70:d1:ee:ff:00" {
			t.Errorf("en1 MAC = %q, want 36:70:d1:ee:ff:00", en1.MAC)
		}
	} else {
		t.Error("en1 not found")
	}

	// vmnet0
	if vmnet, ok := byName["vmnet0"]; ok {
		if vmnet.IP != "192.168.186.1" {
			t.Errorf("vmnet0 IP = %q, want 192.168.186.1", vmnet.IP)
		}
	} else {
		t.Error("vmnet0 not found")
	}

	// utun0
	if utun, ok := byName["utun0"]; ok {
		if utun.IP != "100.64.0.5" {
			t.Errorf("utun0 IP = %q, want 100.64.0.5", utun.IP)
		}
	} else {
		t.Error("utun0 not found")
	}

	// bridge0 (no UP flag in flags=8822<BROADCAST,SMART,SIMPLEX,MULTICAST>)
	if br, ok := byName["bridge0"]; ok {
		if br.State != "down" {
			t.Errorf("bridge0 State = %q, want down", br.State)
		}
	} else {
		t.Error("bridge0 not found")
	}
}

func TestParseIPAddr(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/ip_addr_linux.txt")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	interfaces := ParseIPAddr(string(data))

	if len(interfaces) == 0 {
		t.Fatal("expected interfaces, got none")
	}

	byName := make(map[string]InterfaceInfo)
	for _, iface := range interfaces {
		byName[iface.Name] = iface
	}

	// lo
	if lo, ok := byName["lo"]; ok {
		if lo.IP != "127.0.0.1" {
			t.Errorf("lo IP = %q, want 127.0.0.1", lo.IP)
		}
		if lo.MTU != 65536 {
			t.Errorf("lo MTU = %d, want 65536", lo.MTU)
		}
	} else {
		t.Error("lo not found")
	}

	// eth0
	if eth0, ok := byName["eth0"]; ok {
		if eth0.IP != "10.0.0.50" {
			t.Errorf("eth0 IP = %q, want 10.0.0.50", eth0.IP)
		}
		if eth0.MAC != "dc:a6:32:aa:bb:cc" {
			t.Errorf("eth0 MAC = %q, want dc:a6:32:aa:bb:cc", eth0.MAC)
		}
		if eth0.MTU != 1500 {
			t.Errorf("eth0 MTU = %d, want 1500", eth0.MTU)
		}
		if eth0.IPv6 != "2001:db8::50" {
			t.Errorf("eth0 IPv6 = %q, want 2001:db8::50", eth0.IPv6)
		}
		if eth0.State != "up" {
			t.Errorf("eth0 State = %q, want up", eth0.State)
		}
	} else {
		t.Error("eth0 not found")
	}

	// cali12345abc (@ stripped)
	if cali, ok := byName["cali12345abc"]; ok {
		if cali.MTU != 1450 {
			t.Errorf("cali MTU = %d, want 1450", cali.MTU)
		}
		if cali.State != "up" {
			t.Errorf("cali State = %q, want up", cali.State)
		}
	} else {
		t.Error("cali12345abc not found in parsed interfaces")
	}

	// flannel.1
	if fl, ok := byName["flannel.1"]; ok {
		if fl.IP != "10.42.0.0" {
			t.Errorf("flannel IP = %q, want 10.42.0.0", fl.IP)
		}
	} else {
		t.Error("flannel.1 not found")
	}
}

func TestParseIPAddrJSON(t *testing.T) {
	addrs := []IPAddrJSON{
		{
			IfName:    "eth0",
			Mtu:       1500,
			Operstate: "UP",
			Address:   "aa:bb:cc:dd:ee:ff",
			AddrInfo: []IPAddrInfo{
				{Family: "inet", Local: "10.0.0.1", Prefixlen: 24},
				{Family: "inet6", Local: "fe80::1", Scope: "link"},
				{Family: "inet6", Local: "2001:db8::1", Scope: "global"},
			},
		},
		{
			IfName:    "lo",
			Mtu:       65536,
			Operstate: "UNKNOWN",
			Address:   "00:00:00:00:00:00",
			AddrInfo: []IPAddrInfo{
				{Family: "inet", Local: "127.0.0.1", Prefixlen: 8},
			},
		},
	}

	interfaces := ParseIPAddrJSON(addrs)

	if len(interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(interfaces))
	}

	eth := interfaces[0]
	if eth.Name != "eth0" {
		t.Errorf("name = %q, want eth0", eth.Name)
	}
	if eth.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want 10.0.0.1", eth.IP)
	}
	if eth.IPv6 != "2001:db8::1" {
		t.Errorf("IPv6 = %q, want 2001:db8::1", eth.IPv6)
	}
	if eth.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %q, want aa:bb:cc:dd:ee:ff", eth.MAC)
	}
}
