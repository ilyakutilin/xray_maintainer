package utils

import (
	"fmt"
	"testing"
)

func TestIsValidEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		isValid bool
	}{
		{"valid URL:port", "engage.cloudflareclient.com:2408", true},
		{"valid IPv4:port", "162.159.192.1:2408", true},
		{"valid IPv6:port v1", "[2606:4700:d0::a29f:c001]:2408", true},
		{"valid IPv6:port v2", "[::1]:8080", true},
		{"valid example URL:port", "example.com:8080", true},
		{"valid hyphenated", "valid-domain.com:123", true},
		{"valid localhost:port", "localhost:3000", true},
		{"invalid address", "not a valid address", false},
		{"missing string port", "missing.port", false},
		{"invalid port (too high)", "192.168.1.1:65536", false},
		{"invalid port (string)", "invalid.port:notanumber", false},
		{"invalid IP", "256.256.256.256:80", false},
		{"invalid domain", "invalid..domain:80", false},
		{"invalid (underscores not allowed)", "under_scores.com:80", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEndpoint(tt.addr)
			// TODO: The below shall be handled by assertCorrectBool
			// In order for this to work, test helpers shall be moved to another package
			// assertCorrectBool(t, tt.isValid, result)
			if result != tt.isValid {
				t.Errorf("Test failed for %s: expected %v, got %v", tt.addr, tt.isValid, result)
			} else {
				fmt.Printf("Test passed for %s: %v\n", tt.addr, result)
			}
		})
	}
}
