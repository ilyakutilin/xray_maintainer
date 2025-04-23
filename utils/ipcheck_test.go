package utils

import (
	"fmt"
	"testing"
)

func TestIsExternalIPv4(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		isValid bool
	}{
		{"Google DNS - valid external", "8.8.8.8", true},
		{"Cloudflare DNS - valid external", "1.1.1.1", true},
		{"Private 192.168.x.x", "192.168.1.1", false},
		{"Private 10.x.x.x", "10.0.0.1", false},
		{"Private 172.16.x.x", "172.16.0.1", false},
		{"Loopback", "127.0.0.1", false},
		{"Link-local", "169.254.1.1", false},
		{"Multicast", "224.0.0.1", false},
		{"0.0.0.0", "0.0.0.0", false},
		{"localhost", "localhost", false},
		{"Invalid - not 8 bit", "256.256.256.256", false},
		{"Invalid - not IP", "not.an.ip", false},
		// {"TEST-NET-3", "203.0.113.1", false},
		// {"TEST-NET-2", "198.51.100.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExternalIPv4(tt.ip)
			// TODO: The below shall be handled by assertCorrectBool
			// In order for this to work, test helpers shall be moved to another package
			// assertCorrectBool(t, tt.isValid, result)
			if result != tt.isValid {
				t.Errorf("Test failed for %s: expected %v, got %v", tt.ip, tt.isValid, result)
			} else {
				fmt.Printf("Test passed for %s: %v\n", tt.ip, result)
			}
		})
	}
}
