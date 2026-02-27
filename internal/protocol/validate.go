package protocol

import (
	"fmt"
	"regexp"
)

// Allowed target types.
var allowedTargetTypes = map[string]bool{
	"host": true, "lima": true, "docker": true, "k8s-pod": true,
}

// Allowed shells (exact paths).
var allowedShells = map[string]bool{
	"/bin/bash": true, "/bin/sh": true, "/bin/zsh": true, "": true,
}

// Allowed docker runtimes.
var allowedRuntimes = map[string]bool{
	"docker": true, "podman": true, "": true,
}

// containerNameRe matches valid container/pod/VM names.
var containerNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

const maxNameLen = 253

// ValidateTerminalTarget checks that all fields contain safe, expected values.
func ValidateTerminalTarget(t *TerminalTarget) error {
	if t == nil {
		return nil
	}

	if !allowedTargetTypes[t.Type] {
		return fmt.Errorf("invalid target type: %q", t.Type)
	}

	if !allowedShells[t.Shell] {
		return fmt.Errorf("invalid shell: %q", t.Shell)
	}

	if !allowedRuntimes[t.Runtime] {
		return fmt.Errorf("invalid runtime: %q", t.Runtime)
	}

	for field, val := range map[string]string{
		"container": t.Container,
		"pod":       t.Pod,
		"namespace": t.Namespace,
		"name":      t.Name,
	} {
		if val == "" {
			continue
		}
		if len(val) > maxNameLen {
			return fmt.Errorf("%s name too long (%d chars, max %d)", field, len(val), maxNameLen)
		}
		if !containerNameRe.MatchString(val) {
			return fmt.Errorf("invalid %s name: %q", field, val)
		}
	}

	return nil
}
