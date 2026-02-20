package scanner

import "strings"

// trimOutput removes whitespace from command output.
func trimOutput(out []byte) string {
	return strings.TrimSpace(string(out))
}
