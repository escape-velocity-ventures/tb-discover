package iot

import (
	"context"
	"encoding/json"
	"testing"
)

func TestClassifyDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected DeviceType
	}{
		{"light", TypeLight},
		{"switch", TypeSwitch},
		{"climate", TypeThermostat},
		{"lock", TypeLock},
		{"camera", TypeCamera},
		{"binary_sensor", TypeSensor},
		{"sensor", TypeSensor},
		{"media_player", TypeMedia},
		{"vacuum", TypeVacuum},
		{"cover", TypeCover},
		{"fan", TypeFan},
		{"alarm_control_panel", TypeAlarm},
		{"humidifier", TypeAppliance},
		{"water_heater", TypeAppliance},
		{"unknown_domain", TypeUnknown},
		{"", TypeUnknown},
		{"automation", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := ClassifyDomain(tt.domain)
			if got != tt.expected {
				t.Errorf("ClassifyDomain(%q) = %q, want %q", tt.domain, got, tt.expected)
			}
		})
	}
}

func TestShouldSkipDomain(t *testing.T) {
	skipped := []string{
		"automation", "script", "scene", "zone", "person", "group",
		"input_boolean", "input_number", "input_select", "input_text",
		"timer", "counter", "sun", "weather", "update", "button",
		"number", "select", "text", "tts", "stt", "conversation",
		"schedule", "todo", "calendar", "date", "time", "datetime",
		"event", "image", "persistent_notification", "input_datetime",
	}

	for _, domain := range skipped {
		t.Run("skip_"+domain, func(t *testing.T) {
			if !shouldSkipDomain(domain) {
				t.Errorf("shouldSkipDomain(%q) = false, want true", domain)
			}
		})
	}

	kept := []string{
		"light", "switch", "climate", "lock", "camera", "sensor",
		"binary_sensor", "media_player", "vacuum", "cover", "fan",
	}

	for _, domain := range kept {
		t.Run("keep_"+domain, func(t *testing.T) {
			if shouldSkipDomain(domain) {
				t.Errorf("shouldSkipDomain(%q) = true, want false", domain)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		entityID string
		expected string
	}{
		{"light.kitchen", "light"},
		{"sensor.temperature_bedroom", "sensor"},
		{"climate.living_room", "climate"},
		{"binary_sensor.front_door", "binary_sensor"},
		{"nodot", "nodot"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.entityID, func(t *testing.T) {
			got := extractDomain(tt.entityID)
			if got != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.entityID, got, tt.expected)
			}
		})
	}
}

func TestFilterAttributes(t *testing.T) {
	attrs := map[string]interface{}{
		"friendly_name":       "Kitchen Light",
		"brightness":          200,
		"device_class":        "light",
		"icon":                "mdi:lightbulb",       // should be filtered out
		"supported_features":  63,                     // should be filtered out
		"entity_picture":      "/api/camera_proxy/x",  // should be filtered out
		"manufacturer":        "Philips",
		"model":               "LCT001",
		"battery":             85,
	}

	filtered := filterAttributes(attrs)

	// Kept
	for _, key := range []string{"friendly_name", "brightness", "device_class", "manufacturer", "model", "battery"} {
		if _, ok := filtered[key]; !ok {
			t.Errorf("filterAttributes should keep %q", key)
		}
	}

	// Removed
	for _, key := range []string{"icon", "supported_features", "entity_picture"} {
		if _, ok := filtered[key]; ok {
			t.Errorf("filterAttributes should remove %q", key)
		}
	}
}

func TestFilterAttributesEmpty(t *testing.T) {
	if got := filterAttributes(nil); got != nil {
		t.Errorf("filterAttributes(nil) = %v, want nil", got)
	}

	if got := filterAttributes(map[string]interface{}{"icon": "mdi:x"}); got != nil {
		t.Errorf("filterAttributes with no kept keys should return nil")
	}
}

func TestDeviceJSONShape(t *testing.T) {
	d := Device{
		ID:     "light.kitchen",
		Name:   "Kitchen Light",
		Type:   TypeLight,
		State:  "on",
		Source: "homeassistant",
		Area:   "Kitchen",
		Attributes: map[string]interface{}{
			"brightness": 200,
		},
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"id", "name", "type", "state", "source", "area", "attributes"} {
		if _, ok := m[key]; !ok {
			t.Errorf("Device JSON missing key %q", key)
		}
	}

	if m["type"] != "iot/light" {
		t.Errorf("type = %v, want iot/light", m["type"])
	}
}

func TestDiscoveryResultJSONShape(t *testing.T) {
	r := DiscoveryResult{
		Providers: []string{"homeassistant", "mdns"},
		Devices: []Device{
			{ID: "light.kitchen", Name: "Kitchen", Type: TypeLight, State: "on", Source: "homeassistant"},
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, ok := m["providers"]; !ok {
		t.Error("DiscoveryResult missing providers")
	}
	if _, ok := m["devices"]; !ok {
		t.Error("DiscoveryResult missing devices")
	}
}

func TestSanitizeMDNSName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Living Room Speaker", "living-room-speaker"},
		{"My-Device", "my-device"},
		{"test123", "test123"},
		{"a b@c#d", "a-b-c-d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeMDNSName(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeMDNSName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	devices := []Device{
		{ID: "mdns-hap-device1", Name: "Device 1"},
		{ID: "mdns-hap-device1", Name: "Device 1 Duplicate"},
		{ID: "mdns-hap-device2", Name: "Device 2"},
	}

	result := dedup(devices)
	if len(result) != 2 {
		t.Errorf("dedup returned %d devices, want 2", len(result))
	}
	if result[0].Name != "Device 1" {
		t.Errorf("dedup kept wrong entry: %q", result[0].Name)
	}
}

func TestClassifyUniFiDevice(t *testing.T) {
	tests := []struct {
		oui, name string
		expected  DeviceType
	}{
		{"Ring Inc", "doorbell", TypeCamera},
		{"Google", "Nest Thermostat", TypeThermostat},
		{"Ecobee", "Smart Thermostat", TypeThermostat},
		{"Signify", "Hue Bulb", TypeLight},
		{"Sonos", "One", TypeMedia},
		{"Roku", "Streaming Stick", TypeMedia},
		{"Apple", "Apple TV", TypeMedia},
		{"Google", "Chromecast", TypeMedia},
		{"Samsung", "Smart TV", TypeMedia},
		{"LG", "OLED TV", TypeMedia},
		{"Yale", "Smart Lock", TypeLock},
		{"", "IP Camera", TypeCamera},
		{"Unknown", "laptop", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.oui+"_"+tt.name, func(t *testing.T) {
			got := classifyUniFiDevice(tt.oui, tt.name)
			if got != tt.expected {
				t.Errorf("classifyUniFiDevice(%q, %q) = %q, want %q", tt.oui, tt.name, got, tt.expected)
			}
		})
	}
}

func TestRegistryScanNoProviders(t *testing.T) {
	// With no env vars set, no providers should detect
	reg := NewRegistry()
	result := reg.Scan(context.Background())

	// mDNS might detect on macOS (dns-sd always exists), but HA/Hue/UniFi won't
	// Just verify the result shape is valid
	if result.Devices == nil {
		// nil is ok, means no devices found
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(result): %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("result JSON invalid: %v", err)
	}
}

func TestDeviceTypeConstants(t *testing.T) {
	types := []DeviceType{
		TypeLight, TypeSwitch, TypeThermostat, TypeLock, TypeCamera,
		TypeSensor, TypeMedia, TypeAppliance, TypeCover, TypeFan,
		TypeVacuum, TypeAlarm, TypeUnknown,
	}

	for _, dt := range types {
		if dt == "" {
			t.Error("DeviceType constant should not be empty")
		}
		if dt[:4] != "iot/" {
			t.Errorf("DeviceType %q should start with 'iot/'", dt)
		}
	}
}
