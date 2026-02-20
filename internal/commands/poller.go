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

// Poller fetches approved commands from the SaaS.
type Poller struct {
	url        string
	token      string
	anonKey    string
	httpClient *http.Client
	log        *slog.Logger
}

// NewPoller creates a new command poller.
func NewPoller(baseURL, token, anonKey string) *Poller {
	return &Poller{
		url:     baseURL,
		token:   token,
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: slog.Default().With("component", "command-poller"),
	}
}

// Poll fetches approved commands from the SaaS. Returns empty slice on 404.
func (p *Poller) Poll(ctx context.Context) ([]Command, error) {
	body, err := json.Marshal(PollRequest{AgentToken: p.token})
	if err != nil {
		return nil, fmt.Errorf("marshal poll request: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/cluster-commands/poll", p.url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.anonKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.anonKey)
		httpReq.Header.Set("apikey", p.anonKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Function not deployed yet — not an error
		return nil, nil
	}

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("poll failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result PollResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(result.Commands) > 0 {
		p.log.Info("commands polled", "count", len(result.Commands))
	}
	return result.Commands, nil
}
