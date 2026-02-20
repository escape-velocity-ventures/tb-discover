package ssh

import (
	"regexp"
	"strings"
)

// allowedPrefixes are read-only command prefixes that can be executed remotely.
var allowedPrefixes = []string{
	// System info
	"uname", "hostname", "whoami", "id", "arch",
	"sysctl", "sw_vers", "uptime", "vm_stat", "top -l 1",

	// Hardware / resources
	"cat /proc/cpuinfo", "cat /proc/meminfo",
	"cat /etc/os-release", "cat /etc/machine-id", "cat /etc/hostname",
	"cat /etc/rancher", "cat /var/lib/rancher",
	"cat ~/.kube/config", "cat ~/.lima",
	"df", "free", "lscpu", "nproc",

	// Storage / block devices
	"diskutil list", "diskutil info", "diskutil apfs list",
	"lsblk", "blkid", "mount",

	// Network
	"ip addr", "ip link", "ip route", "ip -j",
	"ifconfig",
	"networksetup -listallhardwareports", "networksetup -getinfo",
	"netstat -rn", "netstat -tlnp", "netstat -ulnp",
	"ss -tlnp", "ss -ulnp",

	// USB enumeration
	"system_profiler SPUSBDataType",
	"ioreg -p IOUSB",

	// Services
	"systemctl list-units", "systemctl status", "systemctl is-active",
	"brew services list",
	"launchctl list",
	"rc-status",

	// Process listing
	"ps aux", "ps -eo",

	// Container runtimes
	"docker ps", "docker info", "docker version", "docker images",
	"podman ps", "podman info", "podman version", "podman images", "podman machine list",
	"nerdctl ps", "nerdctl info", "nerdctl version", "nerdctl images",
	"containerd --version",
	"crictl version", "crictl images",

	// Lima / VM
	"lima list", "limactl list", "limactl shell",

	// Kubernetes detection
	"k3s --version", "k3s kubectl",
	"kubectl version", "kubectl get", "kubectl describe",
	"kubectl api-resources", "kubectl cluster-info", "kubectl config view",
	"kubeadm version",
	"microk8s status", "minikube status",
	"which kubectl", "which k3s", "which kubeadm", "which microk8s",

	// File listing (never content-modifying)
	"ls", "find", "test -f", "test -d",
	"stat", "readlink", "realpath",

	// Version checks
	"git config",
}

// blockedPatterns match dangerous operations even within allowed commands.
var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\brm\b`),
	regexp.MustCompile(`\bmv\b`),
	regexp.MustCompile(`\bcp\b.*>`),
	regexp.MustCompile(`>`),
	regexp.MustCompile(`\bchmod\b`),
	regexp.MustCompile(`\bchown\b`),
	regexp.MustCompile(`\bmkdir\b`),
	regexp.MustCompile(`\btouch\b`),
	regexp.MustCompile(`\bapt\b`),
	regexp.MustCompile(`\byum\b`),
	regexp.MustCompile(`\bbrew install\b`),
	regexp.MustCompile(`\bsudo\b`),
	regexp.MustCompile(`\bsystemctl\s+(start|stop|restart|enable|disable)\b`),
	regexp.MustCompile(`\bkubectl\s+(apply|delete|patch|edit|exec|port-forward|create|replace|scale)\b`),
	regexp.MustCompile(`\bcurl\b.*-X\s*(POST|PUT|DELETE|PATCH)`),
	regexp.MustCompile(`\bwget\b`),
	regexp.MustCompile(`[|&;` + "`" + `$].*\brm\b`),
}

// IsCommandAllowed checks if a command is safe to execute remotely.
// It must match an allowed prefix AND not contain any blocked patterns.
func IsCommandAllowed(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)

	// Check blocked patterns first (defense in depth)
	for _, pat := range blockedPatterns {
		if pat.MatchString(trimmed) {
			return false
		}
	}

	// Check allowed prefixes
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	return false
}
