package iot

import "context"

// DeviceType maps IoT device categories to node types.
type DeviceType string

const (
	TypeLight       DeviceType = "iot/light"
	TypeSwitch      DeviceType = "iot/switch"
	TypeThermostat  DeviceType = "iot/thermostat"
	TypeLock        DeviceType = "iot/lock"
	TypeCamera      DeviceType = "iot/camera"
	TypeSensor      DeviceType = "iot/sensor"
	TypeMedia       DeviceType = "iot/media"
	TypeAppliance   DeviceType = "iot/appliance"
	TypeCover       DeviceType = "iot/cover"
	TypeFan         DeviceType = "iot/fan"
	TypeVacuum      DeviceType = "iot/vacuum"
	TypeAlarm       DeviceType = "iot/alarm"
	TypeUnknown     DeviceType = "iot/unknown"
)

// Device represents a discovered IoT device.
type Device struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       DeviceType        `json:"type"`
	State      string            `json:"state"`
	Source     string            `json:"source"`
	Area       string            `json:"area,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// DiscoveryResult holds all discovered IoT devices.
type DiscoveryResult struct {
	Providers []string `json:"providers"`
	Devices   []Device `json:"devices"`
}

// Provider is implemented by each IoT discovery source.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// Detect checks if this provider is available.
	Detect(ctx context.Context) (bool, error)

	// Discover finds all IoT devices from this source.
	Discover(ctx context.Context) ([]Device, error)
}

// DomainToDeviceType maps Home Assistant entity domains to device types.
var DomainToDeviceType = map[string]DeviceType{
	"light":                TypeLight,
	"switch":               TypeSwitch,
	"climate":              TypeThermostat,
	"lock":                 TypeLock,
	"camera":               TypeCamera,
	"binary_sensor":        TypeSensor,
	"sensor":               TypeSensor,
	"media_player":         TypeMedia,
	"vacuum":               TypeVacuum,
	"cover":                TypeCover,
	"fan":                  TypeFan,
	"alarm_control_panel":  TypeAlarm,
	"humidifier":           TypeAppliance,
	"water_heater":         TypeAppliance,
}

// ClassifyDomain returns the DeviceType for a HA entity domain.
func ClassifyDomain(domain string) DeviceType {
	if dt, ok := DomainToDeviceType[domain]; ok {
		return dt
	}
	return TypeUnknown
}
