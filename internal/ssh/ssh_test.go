package ssh

import (
	"testing"
)

func TestIsCommandAllowed(t *testing.T) {
	allowed := []struct {
		cmd  string
		desc string
	}{
		{"uname -a", "system info"},
		{"hostname", "hostname"},
		{"cat /proc/cpuinfo", "cpu info"},
		{"cat /proc/meminfo", "memory info"},
		{"cat /etc/os-release", "os release"},
		{"df -Pk", "disk free"},
		{"lsblk -J -b", "block devices"},
		{"ip addr show", "ip addresses"},
		{"ip -j addr show", "ip json"},
		{"ip -j route show", "ip routes json"},
		{"ifconfig -a", "ifconfig"},
		{"netstat -rn", "routes"},
		{"docker ps --format json", "docker ps"},
		{"docker info --format json", "docker info"},
		{"podman ps --format json", "podman ps"},
		{"nerdctl ps --format json", "nerdctl ps"},
		{"kubectl get nodes -o json", "kubectl get"},
		{"kubectl describe node foo", "kubectl describe"},
		{"k3s kubectl get pods -A", "k3s kubectl"},
		{"systemctl list-units --type=service", "systemd units"},
		{"systemctl status kubelet", "systemd status"},
		{"ls /etc/rancher", "ls dir"},
		{"free -b", "free memory"},
		{"sysctl -n hw.memsize", "sysctl"},
		{"sw_vers -productVersion", "sw_vers"},
		{"diskutil list", "diskutil"},
		{"ps aux", "process list"},
		{"which kubectl", "which"},
		{"test -f /usr/local/bin/k3s", "test file"},
		{"find /etc/rancher -name config.yaml", "find file"},
	}

	for _, tc := range allowed {
		t.Run(tc.desc, func(t *testing.T) {
			if !IsCommandAllowed(tc.cmd) {
				t.Errorf("expected allowed: %q", tc.cmd)
			}
		})
	}
}

func TestIsCommandBlocked(t *testing.T) {
	blocked := []struct {
		cmd  string
		desc string
	}{
		{"rm -rf /", "rm"},
		{"mv /etc/hosts /tmp", "mv"},
		{"echo foo > /etc/hosts", "redirect"},
		{"chmod 777 /etc/shadow", "chmod"},
		{"chown root:root /tmp/foo", "chown"},
		{"mkdir /tmp/evil", "mkdir"},
		{"touch /tmp/foo", "touch"},
		{"apt install curl", "apt"},
		{"yum install wget", "yum"},
		{"brew install node", "brew install"},
		{"sudo rm -rf /", "sudo"},
		{"systemctl start nginx", "systemctl start"},
		{"systemctl restart kubelet", "systemctl restart"},
		{"kubectl delete pod foo", "kubectl delete"},
		{"kubectl apply -f foo.yaml", "kubectl apply"},
		{"kubectl exec -it pod -- sh", "kubectl exec"},
		{"curl -X POST http://example.com", "curl post"},
		{"wget http://example.com/malware", "wget"},
		{"cat /etc/passwd | rm -rf /", "chained rm"},
		{"ls; rm -rf /", "semicolon rm"},
		{"python3 -c 'import os'", "arbitrary code"},
		{"bash -c 'echo pwned'", "bash exec"},
	}

	for _, tc := range blocked {
		t.Run(tc.desc, func(t *testing.T) {
			if IsCommandAllowed(tc.cmd) {
				t.Errorf("expected blocked: %q", tc.cmd)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input    string
		wantUser string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"root@192.168.1.1", "root", "192.168.1.1", "22", false},
		{"ubuntu@myhost.local", "ubuntu", "myhost.local", "22", false},
		{"user@host:2222", "user", "host", "2222", false},
		{"deploy@[::1]:22", "deploy", "::1", "22", false},
		{"noatsign", "", "", "", true},
		{"@nouser", "", "", "", true},
		{"nohost@", "", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			target, err := ParseTarget(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if target.User != tc.wantUser {
				t.Errorf("user: got %q, want %q", target.User, tc.wantUser)
			}
			if target.Host != tc.wantHost {
				t.Errorf("host: got %q, want %q", target.Host, tc.wantHost)
			}
			if target.Port != tc.wantPort {
				t.Errorf("port: got %q, want %q", target.Port, tc.wantPort)
			}
		})
	}
}

func TestParseTargets(t *testing.T) {
	tests := []struct {
		input string
		count int
	}{
		{"root@host1,ubuntu@host2", 2},
		{"root@host1, ubuntu@host2, deploy@host3", 3},
		{"root@host1", 1},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			targets, err := ParseTargets(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(targets) != tc.count {
				t.Errorf("got %d targets, want %d", len(targets), tc.count)
			}
		})
	}
}

func TestParseTargetsEmpty(t *testing.T) {
	_, err := ParseTargets("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestTargetString(t *testing.T) {
	t1 := Target{User: "root", Host: "example.com", Port: "22"}
	if s := t1.String(); s != "root@example.com" {
		t.Errorf("got %q, want %q", s, "root@example.com")
	}

	t2 := Target{User: "root", Host: "example.com", Port: "2222"}
	if s := t2.String(); s != "root@example.com:2222" {
		t.Errorf("got %q, want %q", s, "root@example.com:2222")
	}
}

func TestTargetAddr(t *testing.T) {
	target := Target{User: "root", Host: "192.168.1.1", Port: "22"}
	if a := target.Addr(); a != "192.168.1.1:22" {
		t.Errorf("got %q, want %q", a, "192.168.1.1:22")
	}
}
