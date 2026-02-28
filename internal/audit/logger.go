package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// AuditLogger writes append-only, hash-chained audit entries to a JSON-lines file.
type AuditLogger struct {
	mu       sync.Mutex
	file     *os.File
	prevHash string
}

// DefaultPath returns the platform-appropriate default audit log path.
func DefaultPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".tb-manage", "audit.log")
	}
	return "/var/log/tb-manage/audit.log"
}

// NewAuditLogger opens (or creates) the audit log file at path.
// The directory is created with 0700; the file with 0600.
// It reads existing entries to recover the last hash for chain continuity.
func NewAuditLogger(path string) (*AuditLogger, error) {
	if path == "" {
		path = DefaultPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("audit: create dir %s: %w", dir, err)
	}

	// Recover previous hash from existing file
	prevHash := ""
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		// Find last non-empty line
		lines := splitLines(data)
		for i := len(lines) - 1; i >= 0; i-- {
			if len(lines[i]) == 0 {
				continue
			}
			var entry AuditEntry
			if json.Unmarshal(lines[i], &entry) == nil {
				prevHash = entry.EntryHash
			}
			break
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit: open %s: %w", path, err)
	}

	return &AuditLogger{file: f, prevHash: prevHash}, nil
}

// Log writes an audit entry, computing its hash chain value.
func (l *AuditLogger) Log(entry AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Compute hash: SHA256(prevHash + json_without_hash)
	entry.EntryHash = "" // clear before hashing
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: marshal: %w", err)
	}

	h := sha256.Sum256(append([]byte(l.prevHash), raw...))
	entry.EntryHash = fmt.Sprintf("%x", h)
	l.prevHash = entry.EntryHash

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: marshal final: %w", err)
	}
	line = append(line, '\n')

	if _, err := l.file.Write(line); err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}
	return nil
}

// Close closes the underlying file.
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// splitLines splits data into JSON-lines (byte slices).
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
