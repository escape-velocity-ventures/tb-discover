package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Completer reports command completion back to the SaaS.
type Completer struct {
	url        string
	token      string
	anonKey    string
	httpClient *http.Client
	log        *slog.Logger
}

// NewCompleter creates a new command completer.
func NewCompleter(baseURL, token, anonKey string) *Completer {
	return &Completer{
		url:     baseURL,
		token:   token,
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: slog.Default().With("component", "command-completer"),
	}
}

// Complete reports a command's result back to the SaaS.
func (c *Completer) Complete(ctx context.Context, cmdID string, result CommandResult) error {
	status := StatusCompleted
	var errMsg string
	if !result.Success {
		status = StatusFailed
		errMsg = result.Message
	}

	req := CompleteRequest{
		AgentToken:   c.token,
		CommandID:    cmdID,
		Status:       status,
		Result:       &result,
		ErrorMessage: errMsg,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal complete request: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/cluster-commands/complete", c.url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.anonKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.anonKey)
		httpReq.Header.Set("apikey", c.anonKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("complete failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	c.log.Debug("command completed", "id", cmdID, "status", status)
	return nil
}
