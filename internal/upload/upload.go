// Package upload handles POSTing scan results to the edge-ingest endpoint.
package upload

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/escape-velocity-ventures/tb-discover/internal/scanners"
)

// Payload is the JSON body sent to edge-ingest.
type Payload struct {
	AgentToken string              `json:"agent_token"`
	Host       scanners.HostScanResult `json:"host"`
	Meta       Meta                `json:"meta"`
}

// Meta contains metadata about the scan.
type Meta struct {
	Version    string   `json:"version"`
	DurationMs int64    `json:"duration_ms"`
	Phases     []string `json:"phases"`
	SourceHost string   `json:"source_host"`
}

// Result holds the HTTP response details.
type Result struct {
	OK     bool
	Status int
	Body   string
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		// Force HTTP/1.1 — Go's HTTP/2 client can hang on POST requests
		// to Cloudflare-fronted endpoints (Supabase) on macOS.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	},
}

// Send posts scan results to the edge-ingest endpoint.
func Send(ingestURL, agentToken, anonKey string, host scanners.HostScanResult, version string, durationMs int64) Result {
	payload := Payload{
		AgentToken: agentToken,
		Host:       host,
		Meta: Meta{
			Version:    version,
			DurationMs: durationMs,
			Phases:     []string{"host"},
			SourceHost: host.Network.Hostname,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Result{OK: false, Status: 0, Body: fmt.Sprintf("marshal error: %v", err)}
	}

	url := fmt.Sprintf("%s/functions/v1/edge-ingest", ingestURL)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return Result{OK: false, Status: 0, Body: fmt.Sprintf("request error: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+anonKey)
	req.Header.Set("apikey", anonKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{OK: false, Status: 0, Body: fmt.Sprintf("network error: %v", err)}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	return Result{
		OK:     resp.StatusCode >= 200 && resp.StatusCode < 300,
		Status: resp.StatusCode,
		Body:   string(respBody),
	}
}
