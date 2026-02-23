package install

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	systemdUnitPath = "/etc/systemd/system/tb-manage.service"
)

// SystemdUnit generates the systemd unit file content.
func SystemdUnit(binPath string) string {
	return fmt.Sprintf(`[Unit]
Description=TinkerBelle Management Agent
Documentation=https://github.com/tinkerbelle-io/tb-manage
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s daemon --config %s
Restart=always
RestartSec=10
Environment=TB_LOG_LEVEL=info

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=%s
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`, binPath, DefaultConfigFile, DefaultConfigDir)
}

func installSystemd(binPath string) error {
	unit := SystemdUnit(binPath)

	if err := os.WriteFile(systemdUnitPath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := runCommand("systemctl", "enable", ServiceName); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	if err := runCommand("systemctl", "start", ServiceName); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	return nil
}

func uninstallSystemd() error {
	_ = runCommand("systemctl", "stop", ServiceName)
	_ = runCommand("systemctl", "disable", ServiceName)
	_ = os.Remove(systemdUnitPath)
	_ = runCommand("systemctl", "daemon-reload")
	return nil
}

func isSystemdRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", ServiceName)
	return cmd.Run() == nil
}
