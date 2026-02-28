package cmd

import "testing"

func TestValidateGatewayURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		allowInsecure bool
		wantErr       bool
	}{
		{"empty URL is fine", "", false, false},
		{"wss:// always allowed", "wss://gateway.example.com/ws", false, false},
		{"wss:// with allow-insecure", "wss://gateway.example.com/ws", true, false},
		{"ws:// rejected without flag", "ws://localhost:8080/ws", false, true},
		{"ws:// allowed with flag", "ws://localhost:8080/ws", true, false},
		{"http:// rejected", "http://gateway.example.com/ws", false, true},
		{"https:// rejected", "https://gateway.example.com/ws", false, true},
		{"random string rejected", "not-a-url", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGatewayURL(tt.url, tt.allowInsecure)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGatewayURL(%q, %v) error = %v, wantErr %v", tt.url, tt.allowInsecure, err, tt.wantErr)
			}
		})
	}
}
