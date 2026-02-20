package protocol

// Message types
const (
	TypeSessionOpen  = "session.open"
	TypeSessionClose = "session.close"
	TypeSessionReady = "session.ready"
	TypeSessionError = "session.error"
	TypePTYInput     = "pty.input"
	TypePTYOutput    = "pty.output"
	TypePTYResize    = "pty.resize"
	TypeHeartbeat    = "agent.heartbeat"
)

// Envelope is used for initial JSON decode to determine message type
type Envelope struct {
	Type string `json:"type"`
}

type SessionOpenMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	HostID    string `json:"hostId"`
	ClusterID string `json:"clusterId"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
}

type SessionCloseMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
}

type SessionReadyMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
}

type SessionErrorMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Error     string `json:"error"`
}

type PTYInputMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Data      string `json:"data"`
}

type PTYOutputMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Data      string `json:"data"`
}

type PTYResizeMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

type HeartbeatMessage struct {
	Type      string `json:"type"`
	AgentID   string `json:"agentId"`
	ClusterID string `json:"clusterId"`
	Timestamp int64  `json:"timestamp"`
}
