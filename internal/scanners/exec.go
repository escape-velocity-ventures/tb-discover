package scanners

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecResult holds the output of a shell command.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// useNsenter controls whether commands run via nsenter (K8s mode).
var useNsenter bool

// SetNsenter enables nsenter mode for K8s DaemonSet operation.
func SetNsenter(enabled bool) {
	useNsenter = enabled
}

// HostExec runs a command on the host, optionally via nsenter.
func HostExec(command string) ExecResult {
	var cmd *exec.Cmd
	if useNsenter {
		cmd = exec.Command("nsenter", "--target", "1", "--mount", "--", "sh", "-c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := runWithTimeout(cmd, 10*time.Second)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return fmt.Errorf("command timed out after %s", timeout)
	}
}

// ReadFile reads a file from the host filesystem.
func ReadFile(path string) (string, error) {
	if useNsenter {
		result := HostExec(fmt.Sprintf("cat %s", path))
		if result.ExitCode != 0 {
			return "", fmt.Errorf("failed to read %s: %s", path, strings.TrimSpace(result.Stderr))
		}
		return result.Stdout, nil
	}

	cmd := exec.Command("cat", path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
