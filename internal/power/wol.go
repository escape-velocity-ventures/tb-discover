package power

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// WoLProvider sends Wake-on-LAN magic packets.
type WoLProvider struct{}

func NewWoLProvider() *WoLProvider { return &WoLProvider{} }

func (p *WoLProvider) Name() string        { return "wol" }
func (p *WoLProvider) Method() PowerMethod  { return MethodWoL }

// Detect always returns true — WoL is a software-only capability.
func (p *WoLProvider) Detect(ctx context.Context) (bool, error) {
	return true, nil
}

// ListTargets returns empty — WoL targets are configured externally.
func (p *WoLProvider) ListTargets(ctx context.Context) ([]PowerTarget, error) {
	return nil, nil
}

// GetState is not possible with WoL — always returns unknown.
func (p *WoLProvider) GetState(ctx context.Context, targetID string) (PowerState, error) {
	return StateUnknown, nil
}

// Execute sends a WoL magic packet. targetID must be a MAC address.
func (p *WoLProvider) Execute(ctx context.Context, targetID string, action PowerAction) error {
	if action != ActionOn {
		return fmt.Errorf("wol only supports 'on' action")
	}
	return SendMagicPacket(targetID)
}

// BuildMagicPacket creates a WoL magic packet for the given MAC address.
func BuildMagicPacket(mac string) ([]byte, error) {
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address %q: %w", mac, err)
	}

	// Magic packet: 6 bytes of 0xFF followed by MAC address repeated 16 times
	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], hwAddr)
	}
	return packet, nil
}

// SendMagicPacket broadcasts a WoL magic packet for the given MAC address.
func SendMagicPacket(mac string) error {
	// Normalize MAC separators
	mac = strings.ReplaceAll(mac, "-", ":")

	packet, err := BuildMagicPacket(mac)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9,
	})
	if err != nil {
		return fmt.Errorf("dial udp: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("send magic packet: %w", err)
	}
	return nil
}
