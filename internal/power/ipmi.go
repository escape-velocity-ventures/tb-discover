package power

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// IPMIProvider controls power via ipmitool.
type IPMIProvider struct{}

func NewIPMIProvider() *IPMIProvider { return &IPMIProvider{} }

func (p *IPMIProvider) Name() string        { return "ipmi" }
func (p *IPMIProvider) Method() PowerMethod  { return MethodIPMI }

func (p *IPMIProvider) Detect(ctx context.Context) (bool, error) {
	_, err := exec.LookPath("ipmitool")
	return err == nil, nil
}

func (p *IPMIProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	// IPMI manages the local BMC — the target is "self"
	state, _ := p.GetState(ctx, "local")
	return []PowerTarget{
		{
			ID:       "ipmi-local",
			Name:     "Local BMC",
			State:    state,
			Method:   MethodIPMI,
			Provider: p.Name(),
		},
	}, nil
}

func (p *IPMIProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	cmd := exec.CommandContext(ctx, "ipmitool", "power", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return StateUnknown, fmt.Errorf("ipmitool: %w", err)
	}

	output := strings.TrimSpace(string(out))
	if strings.Contains(strings.ToLower(output), "on") {
		return StateOn, nil
	}
	if strings.Contains(strings.ToLower(output), "off") {
		return StateOff, nil
	}
	return StateUnknown, nil
}

func (p *IPMIProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	var args []string
	switch action {
	case ActionOn:
		args = []string{"power", "on"}
	case ActionOff:
		args = []string{"power", "off"}
	case ActionCycle:
		args = []string{"power", "cycle"}
	case ActionReset:
		args = []string{"power", "reset"}
	case ActionStatus:
		args = []string{"power", "status"}
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}

	cmd := exec.CommandContext(ctx, "ipmitool", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipmitool %s: %w (%s)", action, err, strings.TrimSpace(string(out)))
	}
	return nil
}
