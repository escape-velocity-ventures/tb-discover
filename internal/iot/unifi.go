package iot

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

// UniFiProvider discovers network devices via UniFi controller API.
type UniFiProvider struct {
	url      string
	username string
	password string
	site     string
}

func NewUniFiProvider() *UniFiProvider {
	site := os.Getenv("UNIFI_SITE")
	if site == "" {
		site = "default"
	}
	return &UniFiProvider{
		url:      os.Getenv("UNIFI_URL"),
		username: os.Getenv("UNIFI_USERNAME"),
		password: os.Getenv("UNIFI_PASSWORD"),
		site:     site,
	}
}

func (p *UniFiProvider) Name() string { return "unifi" }

func (p *UniFiProvider) Detect(ctx context.Context) (bool, error) {
	return p.url != "" && p.username != "" && p.password != "", nil
}

func (p *UniFiProvider) Discover(ctx context.Context) ([]Device, error) {
	client, err := p.login(ctx)
	if err != nil {
		return nil, fmt.Errorf("unifi login: %w", err)
	}

	// Get all clients (devices connected to UniFi network)
	url := fmt.Sprintf("%s/api/s/%s/stat/sta", p.url, p.site)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unifi sta: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			MAC      string `json:"mac"`
			Hostname string `json:"hostname"`
			Name     string `json:"name"`
			IP       string `json:"ip"`
			IsWired  bool   `json:"is_wired"`
			Network  string `json:"network"`
			OUI      string `json:"oui"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	var devices []Device
	for _, client := range result.Data {
		name := client.Name
		if name == "" {
			name = client.Hostname
		}
		if name == "" {
			name = client.MAC
		}

		deviceType := classifyUniFiDevice(client.OUI, name)
		connType := "wifi"
		if client.IsWired {
			connType = "wired"
		}

		devices = append(devices, Device{
			ID:     "unifi-" + strings.ReplaceAll(client.MAC, ":", ""),
			Name:   name,
			Type:   deviceType,
			State:  "connected",
			Source: "unifi",
			Attributes: map[string]interface{}{
				"mac":        client.MAC,
				"ip":         client.IP,
				"connection": connType,
				"network":    client.Network,
				"oui":        client.OUI,
			},
		})
	}

	return devices, nil
}

func (p *UniFiProvider) login(ctx context.Context) (*http.Client, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, p.username, p.password)
	req, err := http.NewRequestWithContext(ctx, "POST", p.url+"/api/login",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	return client, nil
}

// classifyUniFiDevice guesses device type from OUI manufacturer or name.
func classifyUniFiDevice(oui, name string) DeviceType {
	lower := strings.ToLower(oui + " " + name)

	patterns := []struct {
		match      string
		deviceType DeviceType
	}{
		{"ring", TypeCamera},
		{"nest", TypeThermostat},
		{"ecobee", TypeThermostat},
		{"hue", TypeLight},
		{"sonos", TypeMedia},
		{"roku", TypeMedia},
		{"apple tv", TypeMedia},
		{"chromecast", TypeMedia},
		{"samsung", TypeMedia},
		{"lg", TypeMedia},
		{"camera", TypeCamera},
		{"lock", TypeLock},
	}

	for _, p := range patterns {
		if strings.Contains(lower, p.match) {
			return p.deviceType
		}
	}

	return TypeUnknown
}
