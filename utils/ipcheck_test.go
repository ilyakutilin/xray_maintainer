package utils

import (
	"fmt"
	"testing"
)

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"CIDR IPv4", "172.16.0.2/32", true},
		{"CIDR IPv6", "2606:4700:110:8aa0:c8f9:28e3:42c:7a85/128", true},
		{"IPv4", "1.1.1.1", true},
		{"IPv6", "2606:4700:d0::a29f:c001", true},
		{"invalid IP (string)", "invalid IP", false},
		{"invalid IPv4 (not 8 bit)", "123.456.789.0", false},
		{"invalid CIDR", "1.1.1.1/33", false},
		{"invalid IPv6", "::1/129", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIPOrCIDR(tt.ip)
			// TODO: The below shall be handled by assertCorrectBool
			// In order for this to work, test helpers shall be moved to another package
			// assertCorrectBool(t, tt.expected, result)
			if result != tt.expected {
				t.Errorf("Test failed for %s: expected %v, got %v", tt.ip, tt.expected, result)
			} else {
				fmt.Printf("Test passed for %s: %v\n", tt.ip, result)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Google DNS - valid external", "8.8.8.8", true},
		{"Cloudflare DNS - valid external", "1.1.1.1", true},
		{"localhost", "localhost", false},
		{"Invalid - not 8 bit", "256.256.256.256", false},
		{"Invalid - not IP", "not.an.ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIP(tt.ip)
			// TODO: The below shall be handled by assertCorrectBool
			// In order for this to work, test helpers shall be moved to another package
			// assertCorrectBool(t, tt.expected, result)
			if result != tt.expected {
				t.Errorf("Test failed for %s: expected %v, got %v", tt.ip, tt.expected, result)
			} else {
				fmt.Printf("Test passed for %s: %v\n", tt.ip, result)
			}
		})
	}
}

func TestIsValidVlessPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected bool
	}{
		{"80 - valid", 80, true},
		{"443 - valid", 443, true},
		{"22 - invalid", 22, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidVlessPort(tt.port)
			// TODO: The below shall be handled by assertCorrectBool
			if result != tt.expected {
				t.Errorf("Test failed for %d: expected %v, got %v", tt.port, tt.expected, result)
			} else {
				fmt.Printf("Test passed for %d: %v\n", tt.port, result)
			}
		})
	}
}
