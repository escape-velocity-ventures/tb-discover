package power

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// SmartPlugProvider controls TP-Link Kasa smart plugs via local network.
type SmartPlugProvider struct{}

func NewSmartPlugProvider() *SmartPlugProvider { return &SmartPlugProvider{} }

func (p *SmartPlugProvider) Name() string        { return "smart-plug" }
func (p *SmartPlugProvider) Method() PowerMethod  { return MethodSmartPlug }

// Detect checks for kasa CLI tool or python-kasa.
func (p *SmartPlugProvider) Detect(ctx context.Context) (bool, error) {
	_, err := exec.LookPath("kasa")
	return err == nil, nil
}

// ListTargets discovers Kasa devices via broadcast.
func (p *SmartPlugProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	cmd := exec.CommandContext(ctx, "kasa", "discover", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kasa discover: %w", err)
	}

	var devices map[string]json.RawMessage
	if err := json.Unmarshal(out, &devices); err != nil {
		// kasa discover might output line-by-line, try parsing differently
		return p.parseKasaText(string(out)), nil
	}

	var targets []PowerTarget
	for ip, raw := range devices {
		var info struct {
			Alias   string `json:"alias"`
			IsOn    bool   `json:"is_on"`
			DevType string `json:"dev_type"`
		}
		json.Unmarshal(raw, &info)

		state := StateOff
		if info.IsOn {
			state = StateOn
		}

		targets = append(targets, PowerTarget{
			ID:       "plug-" + sanitizeID(info.Alias),
			Name:     info.Alias,
			State:    state,
			Method:   MethodSmartPlug,
			Address:  ip,
			Provider: p.Name(),
		})
	}
	return targets, nil
}

func (p *SmartPlugProvider) parseKasaText(output string) []PowerTarget {
	var targets []PowerTarget
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Host:") {
			// Basic text parsing fallback
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				targets = append(targets, PowerTarget{
					ID:       "plug-" + sanitizeID(strings.TrimSpace(parts[1])),
					Name:     strings.TrimSpace(parts[1]),
					State:    StateUnknown,
					Method:   MethodSmartPlug,
					Address:  strings.TrimSpace(parts[1]),
					Provider: p.Name(),
				})
			}
		}
	}
	return targets
}

func (p *SmartPlugProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	// Kasa protocol: send encrypted JSON to port 9999
	return StateUnknown, nil
}

func (p *SmartPlugProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	var kasaAction string
	switch action {
	case ActionOn:
		kasaAction = "on"
	case ActionOff:
		kasaAction = "off"
	default:
		return fmt.Errorf("smart plug only supports on/off actions")
	}

	// Try kasa CLI first
	addr := strings.TrimPrefix(targetID, "plug-")
	cmd := exec.CommandContext(ctx, "kasa", "--host", addr, kasaAction)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kasa %s: %w (%s)", kasaAction, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// kasaEncrypt encrypts a TP-Link Kasa protocol message.
func kasaEncrypt(plaintext string) []byte {
	n := len(plaintext)
	buf := make([]byte, 4+n)
	buf[0] = byte(n >> 24)
	buf[1] = byte(n >> 16)
	buf[2] = byte(n >> 8)
	buf[3] = byte(n)

	key := byte(171)
	for i := 0; i < n; i++ {
		buf[4+i] = plaintext[i] ^ key
		key = buf[4+i]
	}
	return buf
}

// kasaDecrypt decrypts a TP-Link Kasa protocol response.
func kasaDecrypt(ciphertext []byte) string {
	if len(ciphertext) < 4 {
		return ""
	}
	payload := ciphertext[4:]
	result := make([]byte, len(payload))
	key := byte(171)
	for i, b := range payload {
		result[i] = b ^ key
		key = b
	}
	return string(result)
}

// kasaQuery sends a raw query to a Kasa device.
func kasaQuery(ctx context.Context, host, query string) (string, error) {
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", host+":9999")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(kasaEncrypt(query)); err != nil {
		return "", err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return kasaDecrypt(buf[:n]), nil
}

func sanitizeID(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(s))
}
