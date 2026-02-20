package power

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// HypervisorProvider controls VM power via virsh (libvirt).
type HypervisorProvider struct{}

func NewHypervisorProvider() *HypervisorProvider { return &HypervisorProvider{} }

func (p *HypervisorProvider) Name() string        { return "hypervisor" }
func (p *HypervisorProvider) Method() PowerMethod  { return MethodHypervisor }

func (p *HypervisorProvider) Detect(ctx context.Context) (bool, error) {
	_, err := exec.LookPath("virsh")
	return err == nil, nil
}

func (p *HypervisorProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("virsh list: %w", err)
	}

	var targets []PowerTarget
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		state, _ := p.GetState(ctx, name)
		targets = append(targets, PowerTarget{
			ID:       "vm-" + name,
			Name:     name,
			State:    state,
			Method:   MethodHypervisor,
			Provider: p.Name(),
		})
	}
	return targets, nil
}

func (p *HypervisorProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	name := strings.TrimPrefix(targetID, "vm-")
	cmd := exec.CommandContext(ctx, "virsh", "domstate", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return StateUnknown, fmt.Errorf("virsh domstate: %w", err)
	}

	state := strings.TrimSpace(string(out))
	switch state {
	case "running":
		return StateOn, nil
	case "shut off", "paused", "crashed":
		return StateOff, nil
	default:
		return StateUnknown, nil
	}
}

func (p *HypervisorProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	name := strings.TrimPrefix(targetID, "vm-")
	var args []string

	switch action {
	case ActionOn:
		args = []string{"start", name}
	case ActionOff:
		args = []string{"shutdown", name}
	case ActionCycle:
		args = []string{"reboot", name}
	case ActionReset:
		args = []string{"reset", name}
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}

	cmd := exec.CommandContext(ctx, "virsh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("virsh %s %s: %w (%s)", args[0], name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
