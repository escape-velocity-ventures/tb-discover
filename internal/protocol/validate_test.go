package protocol

import (
	"strings"
	"testing"
)

func TestValidateTerminalTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  *TerminalTarget
		wantErr string // empty = no error
	}{
		{"nil target", nil, ""},
		{"valid host", &TerminalTarget{Type: "host"}, ""},
		{"valid docker", &TerminalTarget{Type: "docker", Container: "my-app", Runtime: "docker", Shell: "/bin/bash"}, ""},
		{"valid k8s-pod", &TerminalTarget{Type: "k8s-pod", Pod: "web-0", Namespace: "default", Container: "nginx", Shell: "/bin/sh"}, ""},
		{"valid lima", &TerminalTarget{Type: "lima", Name: "ubuntu", Shell: "/bin/zsh"}, ""},
		{"empty shell is ok", &TerminalTarget{Type: "host", Shell: ""}, ""},

		// Injection: bad target type
		{"bad target type", &TerminalTarget{Type: "python3"}, "invalid target type"},
		{"bad target type path", &TerminalTarget{Type: "/usr/bin/env"}, "invalid target type"},

		// Injection: bad shell
		{"bad shell", &TerminalTarget{Type: "host", Shell: "python3"}, "invalid shell"},
		{"shell injection", &TerminalTarget{Type: "host", Shell: "/bin/bash -c 'curl evil.com'"}, "invalid shell"},
		{"shell path traversal", &TerminalTarget{Type: "host", Shell: "../../etc/passwd"}, "invalid shell"},

		// Injection: bad runtime
		{"bad runtime", &TerminalTarget{Type: "docker", Runtime: "curl"}, "invalid runtime"},
		{"runtime injection", &TerminalTarget{Type: "docker", Runtime: "docker;rm -rf /"}, "invalid runtime"},

		// Injection: bad container name
		{"container injection", &TerminalTarget{Type: "docker", Container: "x;curl evil.com"}, "invalid container name"},
		{"container with spaces", &TerminalTarget{Type: "docker", Container: "my container"}, "invalid container name"},
		{"container too long", &TerminalTarget{Type: "docker", Container: strings.Repeat("a", 254)}, "name too long"},
		{"container starts with hyphen", &TerminalTarget{Type: "docker", Container: "-evil"}, "invalid container name"},

		// Injection: bad pod/namespace
		{"pod injection", &TerminalTarget{Type: "k8s-pod", Pod: "pod$(whoami)", Namespace: "default"}, "invalid pod name"},
		{"namespace injection", &TerminalTarget{Type: "k8s-pod", Pod: "web", Namespace: "ns;evil"}, "invalid namespace name"},

		// Injection: bad lima VM name
		{"lima name injection", &TerminalTarget{Type: "lima", Name: "vm && curl evil"}, "invalid name name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTerminalTarget(tt.target)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
