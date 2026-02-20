package power

import (
	"context"
	"fmt"
	"os"
)

// CloudProvider controls cloud instance power (AWS, GCE).
type CloudProvider struct{}

func NewCloudProvider() *CloudProvider { return &CloudProvider{} }

func (p *CloudProvider) Name() string        { return "cloud" }
func (p *CloudProvider) Method() PowerMethod  { return MethodCloud }

// Detect checks if we're running on a cloud instance by checking metadata.
func (p *CloudProvider) Detect(ctx context.Context) (bool, error) {
	// Check for AWS
	if _, err := os.Stat("/sys/hypervisor/uuid"); err == nil {
		return true, nil
	}
	// Check for GCE
	if _, err := os.Stat("/sys/class/dmi/id/product_name"); err == nil {
		data, _ := os.ReadFile("/sys/class/dmi/id/product_name")
		if len(data) > 0 && (string(data) == "Google Compute Engine\n" || string(data) == "Google\n") {
			return true, nil
		}
	}
	return false, nil
}

func (p *CloudProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	return nil, nil
}

func (p *CloudProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	return StateUnknown, nil
}

func (p *CloudProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	return fmt.Errorf("cloud: not yet implemented (requires AWS/GCE SDK)")
}
