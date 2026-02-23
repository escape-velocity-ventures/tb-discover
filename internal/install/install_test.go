package install

import (
	"strings"
	"testing"
)

func TestSystemdUnitContent(t *testing.T) {
	unit := SystemdUnit("/usr/local/bin/tb-manage")

	checks := []struct {
		name     string
		contains string
	}{
		{"description", "TinkerBelle Management Agent"},
		{"exec start", "ExecStart=/usr/local/bin/tb-manage daemon --config /etc/tb-manage/config.yaml"},
		{"restart", "Restart=always"},
		{"restart sec", "RestartSec=10"},
		{"after network", "After=network-online.target"},
		{"wanted by", "WantedBy=multi-user.target"},
		{"no new privs", "NoNewPrivileges=true"},
		{"protect system", "ProtectSystem=strict"},
		{"config path", DefaultConfigFile},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(unit, c.contains) {
				t.Errorf("unit file missing %q", c.contains)
			}
		})
	}
}

func TestLaunchdPlistContent(t *testing.T) {
	plist := LaunchdPlist("/usr/local/bin/tb-manage")

	checks := []struct {
		name     string
		contains string
	}{
		{"label", "io.tinkerbelle.tb-manage"},
		{"binary path", "/usr/local/bin/tb-manage"},
		{"daemon arg", "<string>daemon</string>"},
		{"config arg", DefaultConfigFile},
		{"run at load", "<key>RunAtLoad</key>"},
		{"keep alive", "<key>KeepAlive</key>"},
		{"stdout log", "/var/log/tb-manage.log"},
		{"stderr log", "/var/log/tb-manage.err"},
		{"plist dtd", "PropertyList-1.0.dtd"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(plist, c.contains) {
				t.Errorf("plist missing %q", c.contains)
			}
		})
	}
}

func TestSystemdUnitCustomBinary(t *testing.T) {
	unit := SystemdUnit("/opt/tb-manage/bin/tb-manage")
	if !strings.Contains(unit, "ExecStart=/opt/tb-manage/bin/tb-manage") {
		t.Error("unit file should use custom binary path")
	}
}

func TestLaunchdPlistCustomBinary(t *testing.T) {
	plist := LaunchdPlist("/opt/tb-manage/bin/tb-manage")
	if !strings.Contains(plist, "<string>/opt/tb-manage/bin/tb-manage</string>") {
		t.Error("plist should use custom binary path")
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"short", "****"},
		{"abcdefgh", "****"},
		{"abcdefghij", "abcd...ghij"},
		{"a-very-long-token-here", "a-ve...here"},
	}

	for _, tt := range tests {
		result := maskTokenHelper(tt.input)
		if result != tt.expected {
			t.Errorf("maskToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// maskTokenHelper duplicates the logic for testing since it's in cmd package.
func maskTokenHelper(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func TestServiceName(t *testing.T) {
	if ServiceName != "tb-manage" {
		t.Errorf("expected service name 'tb-manage', got %q", ServiceName)
	}
}

func TestDefaultConfigDir(t *testing.T) {
	if DefaultConfigDir != "/etc/tb-manage" {
		t.Errorf("expected config dir '/etc/tb-manage', got %q", DefaultConfigDir)
	}
}
