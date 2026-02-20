package insights

import (
	"crypto/sha256"
	"fmt"
)

// Fingerprint generates a stable, deterministic identifier for deduplication.
// Format: sha256("analyzer:kind:namespace:name")[:16]
func MakeFingerprint(analyzer, kind, namespace, name string) string {
	input := fmt.Sprintf("%s:%s:%s:%s", analyzer, kind, namespace, name)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash[:8]) // 16 hex chars
}
