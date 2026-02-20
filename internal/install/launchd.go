package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	launchdLabel = "io.tinkerbelle.tb-discover"
)

// launchdPlistPath returns the plist path. Uses system-wide location if running as root,
// user-level otherwise.
func launchdPlistPath() string {
	if os.Getuid() == 0 {
		return "/Library/LaunchDaemons/" + launchdLabel + ".plist"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

// LaunchdPlist generates the launchd plist file content.
func LaunchdPlist(binPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>--config</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/tb-discover.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/tb-discover.err</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>TB_LOG_LEVEL</key>
        <string>info</string>
    </dict>
</dict>
</plist>
`, launchdLabel, binPath, DefaultConfigFile)
}

func installLaunchd(binPath string) error {
	plist := LaunchdPlist(binPath)
	plistPath := launchdPlistPath()

	dir := filepath.Dir(plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create plist dir: %w", err)
	}

	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	if err := runCommand("launchctl", "load", plistPath); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

func uninstallLaunchd() error {
	plistPath := launchdPlistPath()
	_ = runCommand("launchctl", "unload", plistPath)
	_ = os.Remove(plistPath)
	return nil
}

func isLaunchdRunning() bool {
	cmd := exec.Command("launchctl", "list", launchdLabel)
	return cmd.Run() == nil
}
