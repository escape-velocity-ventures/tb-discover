package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Reporter uploads insights to a SaaS upstream.
type Reporter struct {
	url        string
	token      string
	anonKey    string
	httpClient *http.Client
	log        *slog.Logger

	// Track last-seen fingerprint set to skip unnecessary uploads
	lastFingerprints string
}

// NewReporter creates a new insight reporter.
func NewReporter(baseURL, token, anonKey string) *Reporter {
	return &Reporter{
		url:     baseURL,
		token:   token,
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: slog.Default().With("component", "insights-reporter"),
	}
}

// Report uploads insights to the SaaS endpoint.
// Returns true if the upload was performed, false if skipped (unchanged).
func (r *Reporter) Report(ctx context.Context, insights []ClusterInsight) (bool, error) {
	fps := ActiveFingerprints(insights)

	// Check if fingerprints changed since last sync
	fpsKey := strings.Join(fps, ",")
	if fpsKey == r.lastFingerprints {
		r.log.Debug("insights unchanged, skipping sync", "count", len(insights))
		return false, nil
	}

	req := SyncRequest{
		AgentToken:         r.token,
		Insights:           insights,
		ActiveFingerprints: fps,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return false, fmt.Errorf("marshal insights: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/cluster-insights/sync", r.url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if r.anonKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.anonKey)
		httpReq.Header.Set("apikey", r.anonKey)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("sync failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result SyncResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("parse response: %w", err)
	}

	r.lastFingerprints = fpsKey
	r.log.Info("insights synced", "upserted", result.Upserted, "auto_resolved", result.AutoResolved)
	return true, nil
}

// FingerprintSetKey returns a comparable string for a set of fingerprints.
func FingerprintSetKey(fps []string) string {
	sorted := make([]string, len(fps))
	copy(sorted, fps)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}
