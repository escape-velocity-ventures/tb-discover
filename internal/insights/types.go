package insights

// ClusterInsight represents a detected issue or observation in a k8s cluster.
type ClusterInsight struct {
	Analyzer       string            `json:"analyzer"`
	Category       string            `json:"category"`
	Severity       string            `json:"severity"` // action, warning, suggestion, info
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	TargetKind     string            `json:"target_kind"`
	TargetNS       string            `json:"target_namespace"`
	TargetName     string            `json:"target_name"`
	Fingerprint    string            `json:"fingerprint"`
	ProposedAction string            `json:"proposed_action,omitempty"`
	ProposedParams map[string]any    `json:"proposed_params,omitempty"`
	AutoRemediable bool              `json:"auto_remediable,omitempty"`
}

// SyncRequest is the payload for POST /functions/v1/cluster-insights/sync.
type SyncRequest struct {
	AgentToken         string           `json:"agent_token"`
	Insights           []ClusterInsight `json:"insights"`
	ActiveFingerprints []string         `json:"active_fingerprints"`
}

// SyncResponse is the response from cluster-insights/sync.
type SyncResponse struct {
	Upserted     int `json:"upserted"`
	AutoResolved int `json:"auto_resolved"`
}
