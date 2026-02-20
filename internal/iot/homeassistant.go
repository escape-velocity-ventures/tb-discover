package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// HomeAssistantProvider discovers IoT devices via the HA REST API.
type HomeAssistantProvider struct {
	url   string
	token string
}

func NewHomeAssistantProvider() *HomeAssistantProvider {
	return &HomeAssistantProvider{
		url:   os.Getenv("HA_URL"),
		token: os.Getenv("HA_TOKEN"),
	}
}

func (p *HomeAssistantProvider) Name() string { return "homeassistant" }

func (p *HomeAssistantProvider) Detect(ctx context.Context) (bool, error) {
	if p.url == "" || p.token == "" {
		return false, nil
	}

	// Quick health check
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", p.url+"/api/", nil)
	if err != nil {
		return false, nil
	}
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// haState represents a single Home Assistant entity state.
type haState struct {
	EntityID   string                 `json:"entity_id"`
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

func (p *HomeAssistantProvider) Discover(ctx context.Context) ([]Device, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", p.url+"/api/states", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ha api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ha api %d: %s", resp.StatusCode, string(body))
	}

	var states []haState
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("decode states: %w", err)
	}

	var devices []Device
	for _, s := range states {
		domain := extractDomain(s.EntityID)
		deviceType := ClassifyDomain(domain)

		// Skip non-device entities (automations, scripts, scenes, zones, etc.)
		if shouldSkipDomain(domain) {
			continue
		}

		name := s.EntityID
		if fn, ok := s.Attributes["friendly_name"].(string); ok {
			name = fn
		}

		area := ""
		if a, ok := s.Attributes["area"].(string); ok {
			area = a
		}

		// Filter attributes to relevant ones only
		attrs := filterAttributes(s.Attributes)

		devices = append(devices, Device{
			ID:         s.EntityID,
			Name:       name,
			Type:       deviceType,
			State:      s.State,
			Source:      "homeassistant",
			Area:       area,
			Attributes: attrs,
		})
	}

	return devices, nil
}

// extractDomain gets the domain from an entity_id (e.g., "light.kitchen" → "light").
func extractDomain(entityID string) string {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// shouldSkipDomain returns true for HA domains that aren't physical devices.
func shouldSkipDomain(domain string) bool {
	skip := map[string]bool{
		"automation":    true,
		"script":        true,
		"scene":         true,
		"zone":          true,
		"person":        true,
		"group":         true,
		"input_boolean": true,
		"input_number":  true,
		"input_select":  true,
		"input_text":    true,
		"input_datetime": true,
		"timer":         true,
		"counter":       true,
		"sun":           true,
		"weather":       true,
		"persistent_notification": true,
		"update":        true,
		"button":        true,
		"number":        true,
		"select":        true,
		"text":          true,
		"tts":           true,
		"stt":           true,
		"conversation":  true,
		"schedule":      true,
		"todo":          true,
		"calendar":      true,
		"date":          true,
		"time":          true,
		"datetime":      true,
		"event":         true,
		"image":         true,
	}
	return skip[domain]
}

// filterAttributes keeps only relevant device attributes.
func filterAttributes(attrs map[string]interface{}) map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}

	keep := map[string]bool{
		"friendly_name":  true,
		"device_class":   true,
		"unit_of_measurement": true,
		"brightness":     true,
		"color_temp":     true,
		"temperature":    true,
		"humidity":       true,
		"battery":        true,
		"power":          true,
		"energy":         true,
		"voltage":        true,
		"current":        true,
		"manufacturer":   true,
		"model":          true,
		"sw_version":     true,
	}

	filtered := make(map[string]interface{})
	for k, v := range attrs {
		if keep[k] {
			filtered[k] = v
		}
	}

	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
