package power

import (
	"context"
	"fmt"
	"os/exec"
)

// PoEProvider controls Power over Ethernet via SNMP.
type PoEProvider struct{}

func NewPoEProvider() *PoEProvider { return &PoEProvider{} }

func (p *PoEProvider) Name() string        { return "poe" }
func (p *PoEProvider) Method() PowerMethod  { return MethodPoE }

// Detect checks for snmpset/snmpget tools.
func (p *PoEProvider) Detect(ctx context.Context) (bool, error) {
	_, err := exec.LookPath("snmpset")
	if err != nil {
		return false, nil
	}
	_, err = exec.LookPath("snmpget")
	return err == nil, nil
}

// ListTargets returns empty — PoE targets require switch configuration.
func (p *PoEProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	// PoE targets are discovered via switch SNMP OIDs.
	// Requires configured switch addresses and community strings.
	return nil, nil
}

func (p *PoEProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	return StateUnknown, nil
}

// Execute controls PoE port power via SNMP.
// targetID format: "switch-ip:port" (e.g., "192.168.1.1:gi0/1")
func (p *PoEProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	return fmt.Errorf("poe: not yet configured (requires switch SNMP credentials)")
}
