package scanners

import (
	"testing"
)

func TestParseIPAddr(t *testing.T) {
	output := loadTestData(t, "ip-addr.txt")
	ifaces := parseIPAddr(output)

	// Should skip loopback, keep eth0, wlan0, docker0, br-k8s, tailscale0, veth1234
	if len(ifaces) != 6 {
		t.Fatalf("expected 6 interfaces (no lo), got %d: %+v", len(ifaces), nameList(ifaces))
	}

	// eth0
	eth := findIface(ifaces, "eth0")
	if eth == nil {
		t.Fatal("eth0 not found")
	}
	if eth.IP != "192.168.1.100" {
		t.Errorf("expected eth0 IP 192.168.1.100, got %s", eth.IP)
	}
	if eth.IPv6 != "fd12:3456:789a::100" {
		t.Errorf("expected eth0 IPv6 fd12:3456:789a::100, got %s", eth.IPv6)
	}
	if eth.Type != "ethernet" {
		t.Errorf("expected eth0 type ethernet, got %s", eth.Type)
	}

	// wlan0
	wlan := findIface(ifaces, "wlan0")
	if wlan == nil {
		t.Fatal("wlan0 not found")
	}
	if wlan.IP != "10.0.0.50" {
		t.Errorf("expected wlan0 IP 10.0.0.50, got %s", wlan.IP)
	}
	if wlan.IPv6 != "" {
		t.Errorf("expected wlan0 to have no global IPv6 (only link-local), got %s", wlan.IPv6)
	}
	if wlan.Type != "wifi" {
		t.Errorf("expected wlan0 type wifi, got %s", wlan.Type)
	}

	// docker0 — bridge type
	docker := findIface(ifaces, "docker0")
	if docker == nil {
		t.Fatal("docker0 not found")
	}
	if docker.Type != "bridge" {
		t.Errorf("expected docker0 type bridge, got %s", docker.Type)
	}
	if docker.IP != "172.17.0.1" {
		t.Errorf("expected docker0 IP 172.17.0.1, got %s", docker.IP)
	}

	// br-k8s — bridge
	brk8s := findIface(ifaces, "br-k8s")
	if brk8s == nil {
		t.Fatal("br-k8s not found")
	}
	if brk8s.Type != "bridge" {
		t.Errorf("expected br-k8s type bridge, got %s", brk8s.Type)
	}

	// tailscale0 — tunnel
	ts := findIface(ifaces, "tailscale0")
	if ts == nil {
		t.Fatal("tailscale0 not found")
	}
	if ts.Type != "tunnel" {
		t.Errorf("expected tailscale0 type tunnel, got %s", ts.Type)
	}
	if ts.IP != "100.64.0.5" {
		t.Errorf("expected tailscale0 IP 100.64.0.5, got %s", ts.IP)
	}

	// veth — virtual
	veth := findIface(ifaces, "veth1234@if2")
	if veth == nil {
		t.Fatal("veth1234@if2 not found")
	}
	if veth.Type != "virtual" {
		t.Errorf("expected veth type virtual, got %s", veth.Type)
	}
}

func TestParseIPAddr_Empty(t *testing.T) {
	ifaces := parseIPAddr("")
	if len(ifaces) != 0 {
		t.Errorf("expected 0 interfaces for empty input, got %d", len(ifaces))
	}
}

func TestGuessInterfaceType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"lo", "loopback"},
		{"lo0", "loopback"},
		{"eth0", "ethernet"},
		{"en0", "ethernet"},
		{"wlan0", "wifi"},
		{"wlp3s0", "wifi"},
		{"br0", "bridge"},
		{"docker0", "bridge"},
		{"cni0", "bridge"},
		{"tun0", "tunnel"},
		{"wg0", "tunnel"},
		{"tailscale0", "tunnel"},
		{"veth123", "virtual"},
		{"cali456", "virtual"},
		{"flannel.1", "virtual"},
		{"virbr0", "other"},
	}

	for _, tc := range tests {
		result := guessInterfaceType(tc.name)
		if result != tc.expected {
			t.Errorf("guessInterfaceType(%q) = %q, expected %q", tc.name, result, tc.expected)
		}
	}
}

func TestIsDarwinVirtualInterface(t *testing.T) {
	virtual := []string{"anpi0", "anpi1", "ap1", "awdl0", "bridge0", "gif0", "llw0", "stf0", "utun0", "utun3", "XHC0", "pktap0"}
	for _, name := range virtual {
		if !isDarwinVirtualInterface(name) {
			t.Errorf("expected %q to be virtual", name)
		}
	}

	physical := []string{"en0", "en1", "eth0", "lo0", "wlan0", "tailscale0", "docker0"}
	for _, name := range physical {
		if isDarwinVirtualInterface(name) {
			t.Errorf("expected %q to NOT be virtual", name)
		}
	}
}

func findIface(ifaces []NetworkInterface, name string) *NetworkInterface {
	for i := range ifaces {
		if ifaces[i].Name == name {
			return &ifaces[i]
		}
	}
	return nil
}

func nameList(ifaces []NetworkInterface) []string {
	names := make([]string, len(ifaces))
	for i, iface := range ifaces {
		names[i] = iface.Name
	}
	return names
}
