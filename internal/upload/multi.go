package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// Uploader is the interface for uploading scan results.
type Uploader interface {
	Upload(ctx context.Context, req *EdgeIngestRequest) (*EdgeIngestResponse, error)
}

// MultiClient uploads to multiple upstreams concurrently.
type MultiClient struct {
	upstreams []namedClient
	log       *slog.Logger
}

type namedClient struct {
	name   string
	client *Client
	token  string
}

// NewMultiClient creates an uploader from upstream configs.
func NewMultiClient(upstreams []Upstream) *MultiClient {
	mc := &MultiClient{
		log: slog.Default().With("component", "upload"),
	}
	for _, u := range upstreams {
		mc.upstreams = append(mc.upstreams, namedClient{
			name:   u.Name,
			client: NewClient(u.URL, u.Token, u.AnonKey),
			token:  u.Token,
		})
	}
	return mc
}

// Upload sends scan results to all upstreams. Returns the first successful
// response. Logs errors for individual upstreams but only fails if all fail.
func (mc *MultiClient) Upload(ctx context.Context, req *EdgeIngestRequest) (*EdgeIngestResponse, error) {
	if len(mc.upstreams) == 0 {
		return nil, fmt.Errorf("no upstreams configured")
	}

	// Marshal once, each upstream gets its own copy with the right token
	baseBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	type result struct {
		name string
		resp *EdgeIngestResponse
		err  error
	}

	ch := make(chan result, len(mc.upstreams))

	for _, u := range mc.upstreams {
		go func(u namedClient) {
			// Each upstream gets its own request copy with the correct token
			var upstreamReq EdgeIngestRequest
			json.Unmarshal(baseBody, &upstreamReq)
			upstreamReq.AgentToken = u.token

			resp, err := u.client.Upload(ctx, &upstreamReq)
			ch <- result{name: u.name, resp: resp, err: err}
		}(u)
	}

	var firstResp *EdgeIngestResponse
	var errors []string

	for range mc.upstreams {
		r := <-ch
		if r.err != nil {
			mc.log.Warn("upstream upload failed", "upstream", r.name, "error", r.err)
			errors = append(errors, fmt.Sprintf("%s: %v", r.name, r.err))
		} else {
			mc.log.Info("uploaded", "upstream", r.name,
				"session_id", r.resp.SessionID,
				"cluster_id", r.resp.ClusterID,
				"resources", r.resp.ResourceCount)
			if firstResp == nil {
				firstResp = r.resp
			}
		}
	}

	if firstResp != nil {
		return firstResp, nil
	}
	return nil, fmt.Errorf("all upstreams failed: %v", errors)
}
