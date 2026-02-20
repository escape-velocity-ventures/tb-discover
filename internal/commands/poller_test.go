package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPollerHandles404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	p := NewPoller(server.URL, "test-token", "test-key")
	cmds, err := p.Poll(context.Background())

	if err != nil {
		t.Errorf("404 should not return error, got: %v", err)
	}
	if cmds != nil {
		t.Errorf("404 should return nil commands, got: %v", cmds)
	}
}

func TestPollerReturnsCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request shape
		var req PollRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AgentToken != "test-token" {
			t.Errorf("expected test-token, got %s", req.AgentToken)
		}

		resp := PollResponse{
			Commands: []Command{
				{ID: "cmd-1", Action: "delete_pod", TargetKind: "Pod", TargetNamespace: "default", TargetName: "stale-pod"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewPoller(server.URL, "test-token", "test-key")
	cmds, err := p.Poll(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Action != "delete_pod" {
		t.Errorf("expected delete_pod, got %s", cmds[0].Action)
	}
}

func TestCompleteRequestJSON(t *testing.T) {
	req := CompleteRequest{
		AgentToken: "tok",
		CommandID:  "cmd-1",
		Status:     StatusCompleted,
		Result:     &CommandResult{Success: true, Message: "done"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)

	for _, key := range []string{"agent_token", "command_id", "status", "result"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in JSON", key)
		}
	}
}
