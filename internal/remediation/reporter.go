package remediation

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

// Reporter uploads remediation results to a SaaS upstream.
type Reporter struct {
	url        string
	token      string
	anonKey    string
	httpClient *http.Client
	log        *slog.Logger
}

// NewReporter creates a new remediation reporter.
func NewReporter(baseURL, token, anonKey string) *Reporter {
	return &Reporter{
		url:     baseURL,
		token:   token,
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: slog.Default().With("component", "remediation-reporter"),
	}
}

// Report uploads remediation results to the SaaS endpoint.
func (r *Reporter) Report(ctx context.Context, results []RemediationResult) error {
	if len(results) == 0 {
		return nil
	}

	req := ReportRequest{
		AgentToken:   r.token,
		Remediations: results,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal remediations: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/cluster-remediations/report", r.url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if r.anonKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.anonKey)
		httpReq.Header.Set("apikey", r.anonKey)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("report failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	r.log.Info("remediation results reported", "count", len(results))
	return nil
}
