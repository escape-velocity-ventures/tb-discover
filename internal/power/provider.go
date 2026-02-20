package power

import "context"

// PowerState represents the current power state of a target.
type PowerState string

const (
	StateOn      PowerState = "on"
	StateOff     PowerState = "off"
	StateUnknown PowerState = "unknown"
)

// PowerAction represents a power control action.
type PowerAction string

const (
	ActionOn       PowerAction = "on"
	ActionOff      PowerAction = "off"
	ActionCycle    PowerAction = "cycle"
	ActionReset    PowerAction = "reset"
	ActionStatus   PowerAction = "status"
)

// PowerMethod identifies the power control mechanism.
type PowerMethod string

const (
	MethodIPMI       PowerMethod = "ipmi"
	MethodWoL        PowerMethod = "wol"
	MethodHypervisor PowerMethod = "hypervisor"
	MethodSmartPlug  PowerMethod = "smart-plug"
	MethodPoE        PowerMethod = "poe"
	MethodCloud      PowerMethod = "cloud"
)

// PowerTarget represents something whose power can be controlled.
type PowerTarget struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	State    PowerState  `json:"state"`
	Method   PowerMethod `json:"method"`
	Address  string      `json:"address,omitempty"`
	Provider string      `json:"provider"`
}

// PowerRelationship describes a controller→target power dependency.
type PowerRelationship struct {
	ControllerID string      `json:"controllerId"`
	TargetID     string      `json:"targetId"`
	Method       PowerMethod `json:"method"`
}

// PowerCapabilities summarizes what power control is available.
type PowerCapabilities struct {
	Providers     []string            `json:"providers"`
	Targets       []PowerTarget       `json:"targets"`
	Relationships []PowerRelationship `json:"relationships,omitempty"`
}

// Provider is implemented by each power control mechanism.
type Provider interface {
	// Name returns the provider identifier (e.g., "ipmi", "wol").
	Name() string

	// Method returns the power method type.
	Method() PowerMethod

	// Detect checks if this provider is available on the current system.
	Detect(ctx context.Context) (bool, error)

	// ListTargets returns all power targets this provider can control.
	ListTargets(ctx context.Context) ([]PowerTarget, error)

	// GetState returns the current power state of a target.
	GetState(ctx context.Context, targetID string) (PowerState, error)

	// Execute performs a power action on a target.
	Execute(ctx context.Context, targetID string, action PowerAction) error
}
