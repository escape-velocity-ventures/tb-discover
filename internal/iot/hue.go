package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// HueProvider discovers lights via Philips Hue bridge API.
type HueProvider struct {
	bridgeIP string
	username string
}

func NewHueProvider() *HueProvider {
	return &HueProvider{
		bridgeIP: os.Getenv("HUE_BRIDGE_IP"),
		username: os.Getenv("HUE_USERNAME"),
	}
}

func (p *HueProvider) Name() string { return "hue" }

func (p *HueProvider) Detect(ctx context.Context) (bool, error) {
	if p.bridgeIP == "" || p.username == "" {
		return false, nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://%s/api/%s/config", p.bridgeIP, p.username)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

func (p *HueProvider) Discover(ctx context.Context) ([]Device, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("http://%s/api/%s/lights", p.bridgeIP, p.username)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hue api: %w", err)
	}
	defer resp.Body.Close()

	var lights map[string]struct {
		State struct {
			On        bool `json:"on"`
			Bri       int  `json:"bri"`
			Reachable bool `json:"reachable"`
		} `json:"state"`
		Type             string `json:"type"`
		Name             string `json:"name"`
		ModelID          string `json:"modelid"`
		ManufacturerName string `json:"manufacturername"`
		UniqueID         string `json:"uniqueid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lights); err != nil {
		return nil, fmt.Errorf("decode lights: %w", err)
	}

	var devices []Device
	for id, light := range lights {
		state := "off"
		if light.State.On {
			state = "on"
		}
		if !light.State.Reachable {
			state = "unreachable"
		}

		devices = append(devices, Device{
			ID:     "hue-" + id,
			Name:   light.Name,
			Type:   TypeLight,
			State:  state,
			Source: "hue",
			Attributes: map[string]interface{}{
				"model":        light.ModelID,
				"manufacturer": light.ManufacturerName,
				"type":         light.Type,
				"brightness":   light.State.Bri,
				"unique_id":    light.UniqueID,
			},
		})
	}

	return devices, nil
}
