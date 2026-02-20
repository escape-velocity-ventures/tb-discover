package power

import (
	"context"
	"encoding/json"
	"net"
	"testing"
)

func TestBuildMagicPacket(t *testing.T) {
	packet, err := BuildMagicPacket("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(packet) != 102 {
		t.Fatalf("expected 102 bytes, got %d", len(packet))
	}

	// First 6 bytes should be 0xFF
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Errorf("byte %d: expected 0xFF, got 0x%02X", i, packet[i])
		}
	}

	// MAC should be repeated 16 times
	mac, _ := net.ParseMAC("AA:BB:CC:DD:EE:FF")
	for i := 0; i < 16; i++ {
		offset := 6 + i*6
		for j := 0; j < 6; j++ {
			if packet[offset+j] != mac[j] {
				t.Errorf("repetition %d, byte %d: expected 0x%02X, got 0x%02X",
					i, j, mac[j], packet[offset+j])
			}
		}
	}
}

func TestBuildMagicPacketDashSeparator(t *testing.T) {
	packet, err := BuildMagicPacket("AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(packet) != 102 {
		t.Fatalf("expected 102 bytes, got %d", len(packet))
	}
}

func TestBuildMagicPacketInvalid(t *testing.T) {
	_, err := BuildMagicPacket("not-a-mac")
	if err == nil {
		t.Fatal("expected error for invalid MAC")
	}
}

func TestKasaEncryptDecrypt(t *testing.T) {
	original := `{"system":{"get_sysinfo":{}}}`
	encrypted := kasaEncrypt(original)
	decrypted := kasaDecrypt(encrypted)

	if decrypted != original {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, original)
	}
}

func TestKasaEncryptLength(t *testing.T) {
	msg := `{"system":{"get_sysinfo":{}}}`
	encrypted := kasaEncrypt(msg)

	// 4 bytes header + message length
	if len(encrypted) != 4+len(msg) {
		t.Errorf("expected %d bytes, got %d", 4+len(msg), len(encrypted))
	}
}

func TestPowerCapabilitiesJSON(t *testing.T) {
	caps := PowerCapabilities{
		Providers: []string{"wol", "ipmi"},
		Targets: []PowerTarget{
			{
				ID:       "ipmi-local",
				Name:     "Local BMC",
				State:    StateOn,
				Method:   MethodIPMI,
				Provider: "ipmi",
			},
		},
		Relationships: []PowerRelationship{
			{
				ControllerID: "switch-1",
				TargetID:     "jetson-1",
				Method:       MethodPoE,
			},
		},
	}

	data, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	for _, key := range []string{"providers", "targets", "relationships"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}

	// Target shape
	targets := m["targets"].([]interface{})
	target := targets[0].(map[string]interface{})
	for _, key := range []string{"id", "name", "state", "method", "provider"} {
		if _, ok := target[key]; !ok {
			t.Errorf("target missing key %q", key)
		}
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"My Smart Plug", "my-smart-plug"},
		{"Living Room Light!", "living-room-light-"},
		{"plug_123", "plug-123"},
		{"ALLCAPS", "allcaps"},
	}

	for _, tt := range tests {
		got := sanitizeID(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRegistryScan(t *testing.T) {
	reg := NewRegistry()
	caps := reg.Scan(context.Background())

	// WoL should always be detected
	found := false
	for _, p := range caps.Providers {
		if p == "wol" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected WoL provider to be detected")
	}
}

func TestPowerStateValues(t *testing.T) {
	if StateOn != "on" || StateOff != "off" || StateUnknown != "unknown" {
		t.Error("power state constants have wrong values")
	}
}

func TestPowerMethodValues(t *testing.T) {
	methods := map[PowerMethod]string{
		MethodIPMI:       "ipmi",
		MethodWoL:        "wol",
		MethodHypervisor: "hypervisor",
		MethodSmartPlug:  "smart-plug",
		MethodPoE:        "poe",
		MethodCloud:      "cloud",
	}
	for m, want := range methods {
		if string(m) != want {
			t.Errorf("method %v != %q", m, want)
		}
	}
}
