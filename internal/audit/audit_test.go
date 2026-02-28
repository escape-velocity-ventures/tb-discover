package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLogFileCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "audit.log")

	l, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Check directory permissions
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("dir perm = %o, want 0700", perm)
	}

	// Write an entry so the file exists
	if err := l.Log(AuditEntry{SessionID: "s1", EventType: EventSessionOpen}); err != nil {
		t.Fatal(err)
	}

	// Check file permissions
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file perm = %o, want 0600", perm)
	}
}

func TestAppendOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		if err := l.Log(AuditEntry{
			SessionID: fmt.Sprintf("s%d", i),
			EventType: EventCommand,
			Input:     fmt.Sprintf("cmd%d", i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()

	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	// Filter empty
	var nonEmpty int
	for _, ln := range lines {
		if len(ln) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty != 5 {
		t.Errorf("got %d lines, want 5", nonEmpty)
	}
}

func TestHashChainIntegrity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	entries := []AuditEntry{
		{SessionID: "s1", EventType: EventSessionOpen, Timestamp: time.Now().UTC()},
		{SessionID: "s1", EventType: EventCommand, Input: "ls -la", Timestamp: time.Now().UTC()},
		{SessionID: "s1", EventType: EventSessionClose, Timestamp: time.Now().UTC()},
	}
	for _, e := range entries {
		if err := l.Log(e); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()

	// Verify chain
	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	prevHash := ""
	for i, ln := range lines {
		if len(ln) == 0 {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal(ln, &entry); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		recordedHash := entry.EntryHash

		// Recompute
		entry.EntryHash = ""
		raw, _ := json.Marshal(entry)
		h := sha256.Sum256(append([]byte(prevHash), raw...))
		expected := fmt.Sprintf("%x", h)

		if recordedHash != expected {
			t.Fatalf("line %d: hash mismatch: got %s, want %s", i, recordedHash, expected)
		}
		prevHash = recordedHash
	}
}

func TestHashChainContinuity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	// First logger writes entries
	l1, _ := NewAuditLogger(path)
	l1.Log(AuditEntry{SessionID: "s1", EventType: EventSessionOpen, Timestamp: time.Now().UTC()})
	l1.Log(AuditEntry{SessionID: "s1", EventType: EventCommand, Input: "whoami", Timestamp: time.Now().UTC()})
	l1.Close()

	// Second logger picks up chain
	l2, _ := NewAuditLogger(path)
	l2.Log(AuditEntry{SessionID: "s2", EventType: EventSessionOpen, Timestamp: time.Now().UTC()})
	l2.Close()

	// Verify full chain
	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	prevHash := ""
	count := 0
	for _, ln := range lines {
		if len(ln) == 0 {
			continue
		}
		var entry AuditEntry
		json.Unmarshal(ln, &entry)
		recordedHash := entry.EntryHash
		entry.EntryHash = ""
		raw, _ := json.Marshal(entry)
		h := sha256.Sum256(append([]byte(prevHash), raw...))
		expected := fmt.Sprintf("%x", h)
		if recordedHash != expected {
			t.Fatalf("entry %d: chain broken", count)
		}
		prevHash = recordedHash
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 entries, got %d", count)
	}
}

func TestConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l, err := NewAuditLogger(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			l.Log(AuditEntry{
				SessionID: fmt.Sprintf("s%d", i),
				EventType: EventCommand,
				Input:     fmt.Sprintf("cmd%d", i),
			})
		}(i)
	}
	wg.Wait()

	// All entries written and chain valid
	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	count := 0
	prevHash := ""
	for _, ln := range lines {
		if len(ln) == 0 {
			continue
		}
		var entry AuditEntry
		json.Unmarshal(ln, &entry)
		recordedHash := entry.EntryHash
		entry.EntryHash = ""
		raw, _ := json.Marshal(entry)
		h := sha256.Sum256(append([]byte(prevHash), raw...))
		expected := fmt.Sprintf("%x", h)
		if recordedHash != expected {
			t.Fatalf("entry %d: chain broken under concurrency", count)
		}
		prevHash = recordedHash
		count++
	}
	if count != n {
		t.Errorf("got %d entries, want %d", count, n)
	}
}
