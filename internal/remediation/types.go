package remediation

// Action is a restricted set of operations that can be auto-executed.
type Action string

const (
	ActionDeletePod      Action = "delete_pod"
	ActionForceDeletePod Action = "force_delete_pod"
	ActionDeletePVC      Action = "delete_pvc"
)

// AllowedActions is the complete set of actions that auto-remediation may execute.
var AllowedActions = map[Action]bool{
	ActionDeletePod:      true,
	ActionForceDeletePod: true,
	ActionDeletePVC:      true,
}

// RemediationResult records the outcome of a single remediation attempt.
type RemediationResult struct {
	Action             Action `json:"action"`
	TargetKind         string `json:"target_kind"`
	TargetNamespace    string `json:"target_namespace"`
	TargetName         string `json:"target_name"`
	InsightFingerprint string `json:"insight_fingerprint"`
	Reason             string `json:"reason"`
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	DryRun             bool   `json:"dry_run"`
}

// ReportRequest is the payload for POST /functions/v1/cluster-remediations/report.
type ReportRequest struct {
	AgentToken   string              `json:"agent_token"`
	Remediations []RemediationResult `json:"remediations"`
}
