package audit

import "time"

// EventType constants for audit log entries.
const (
	EventSessionOpen  = "SESSION_OPEN"
	EventSessionClose = "SESSION_CLOSE"
	EventCommand      = "COMMAND"
	EventBlocked      = "BLOCKED"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id,omitempty"`
	Origin    string    `json:"origin,omitempty"`
	Input     string    `json:"input,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	EntryHash string    `json:"entry_hash"`
}
