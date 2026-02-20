package commands

// Command represents an approved command from the SaaS.
type Command struct {
	ID              string         `json:"id"`
	Action          string         `json:"action"`
	TargetKind      string         `json:"target_kind"`
	TargetNamespace string         `json:"target_namespace"`
	TargetName      string         `json:"target_name"`
	Parameters      map[string]any `json:"parameters,omitempty"`
}

// CommandResult is the outcome of executing a command.
type CommandResult struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// CompletionStatus indicates the final state of a command.
type CompletionStatus string

const (
	StatusCompleted CompletionStatus = "completed"
	StatusFailed    CompletionStatus = "failed"
)

// PollRequest is the payload for POST /functions/v1/cluster-commands/poll.
type PollRequest struct {
	AgentToken string `json:"agent_token"`
}

// PollResponse is the response from cluster-commands/poll.
type PollResponse struct {
	Commands []Command `json:"commands"`
}

// CompleteRequest is the payload for POST /functions/v1/cluster-commands/complete.
type CompleteRequest struct {
	AgentToken   string           `json:"agent_token"`
	CommandID    string           `json:"command_id"`
	Status       CompletionStatus `json:"status"`
	Result       *CommandResult   `json:"result,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
}
