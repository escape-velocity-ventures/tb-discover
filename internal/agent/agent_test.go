package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tinkerbelle-io/tb-manage/internal/protocol"
)

func TestHandleMessageRouting(t *testing.T) {
	// Test that envelope parsing correctly identifies message types
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "session.open",
			input:    `{"type":"session.open","sessionId":"s1","hostId":"h1","clusterId":"c1","cols":80,"rows":24}`,
			wantType: protocol.TypeSessionOpen,
		},
		{
			name:     "pty.input",
			input:    `{"type":"pty.input","sessionId":"s1","data":"ls\n"}`,
			wantType: protocol.TypePTYInput,
		},
		{
			name:     "pty.resize",
			input:    `{"type":"pty.resize","sessionId":"s1","cols":120,"rows":40}`,
			wantType: protocol.TypePTYResize,
		},
		{
			name:     "session.close",
			input:    `{"type":"session.close","sessionId":"s1"}`,
			wantType: protocol.TypeSessionClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env protocol.Envelope
			err := json.Unmarshal([]byte(tt.input), &env)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if env.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, env.Type)
			}
		})
	}
}

func TestSessionOpenParsing(t *testing.T) {
	input := `{"type":"session.open","sessionId":"abc-123","hostId":"host-1","clusterId":"cluster-1","cols":120,"rows":40}`
	var msg protocol.SessionOpenMessage
	err := json.Unmarshal([]byte(input), &msg)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.SessionID != "abc-123" {
		t.Errorf("expected sessionId abc-123, got %s", msg.SessionID)
	}
	if msg.Cols != 120 {
		t.Errorf("expected cols 120, got %d", msg.Cols)
	}
	if msg.Rows != 40 {
		t.Errorf("expected rows 40, got %d", msg.Rows)
	}
}

func TestHeartbeatSerialization(t *testing.T) {
	msg := protocol.HeartbeatMessage{
		Type:      protocol.TypeHeartbeat,
		AgentID:   "test-agent",
		ClusterID: "test-cluster",
		Timestamp: 1700000000,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded["type"] != "agent.heartbeat" {
		t.Errorf("expected type agent.heartbeat, got %s", decoded["type"])
	}
	if decoded["agentId"] != "test-agent" {
		t.Errorf("expected agentId test-agent, got %s", decoded["agentId"])
	}
}

func TestPermissionDenied(t *testing.T) {
	// Agent without terminal permission should reject session.open
	a := New(Config{
		Token:       "test",
		IdleTimeout: 30 * time.Minute,
		Permissions: []string{"scan"}, // no "terminal"
	})

	msg := protocol.SessionOpenMessage{
		Type:      protocol.TypeSessionOpen,
		SessionID: "test-session",
		Cols:      80,
		Rows:      24,
	}

	err := a.handleSessionOpen(msg)
	if err == nil {
		t.Fatal("expected error for missing terminal permission")
	}
	if err.Error() != "terminal permission not granted" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMaxSessionLimit(t *testing.T) {
	a := New(Config{
		Token:       "test",
		IdleTimeout: 30 * time.Minute,
		Permissions: []string{"terminal"},
		MaxSessions: 2,
	})

	// MaxSessions is checked inside handleSessionOpen, but spawning PTY
	// would require a real connection. Test the limit logic by checking
	// the config was applied correctly.
	if a.maxSessions != 2 {
		t.Errorf("expected maxSessions 2, got %d", a.maxSessions)
	}
	if !a.permissions["terminal"] {
		t.Error("expected terminal permission")
	}
	if a.permissions["scan"] {
		t.Error("should not have scan permission")
	}
}

func TestDefaultMaxSessions(t *testing.T) {
	a := New(Config{
		Token:       "test",
		IdleTimeout: 30 * time.Minute,
		Permissions: []string{"terminal"},
	})

	if a.maxSessions != DefaultMaxSessions {
		t.Errorf("expected default max sessions %d, got %d", DefaultMaxSessions, a.maxSessions)
	}
}
