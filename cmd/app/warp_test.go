package main

import (
	"errors"
	"testing"
)

func TestParseCFCreds(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    CFCreds
		expectError bool
		errMsg      string
	}{
		{
			name: "complete valid input",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [1, 2, 3]
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expected: CFCreds{
				SecretKey: "sk_test_123",
				PublicKey: "pk_test_456",
				Reserved:  []int{1, 2, 3},
				V4:        "1.2.3.4",
				V6:        "2001:db8::1",
				Endpoint:  "https://example.com",
			},
			expectError: false,
		},
		{
			name: "empty reserved",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: []
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expected: CFCreds{
				SecretKey: "sk_test_123",
				PublicKey: "pk_test_456",
				Reserved:  []int{},
				V4:        "1.2.3.4",
				V6:        "2001:db8::1",
				Endpoint:  "https://example.com",
			},
			expectError: true,
			errMsg:      "missing required field: reserved",
		},
		{
			name: "spaces in reserved",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [ 1 , 2 , 3 ]
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expected: CFCreds{
				SecretKey: "sk_test_123",
				PublicKey: "pk_test_456",
				Reserved:  []int{1, 2, 3},
				V4:        "1.2.3.4",
				V6:        "2001:db8::1",
				Endpoint:  "https://example.com",
			},
			expectError: false,
		},
		{
			name: "missing private_key",
			input: `public_key: pk_test_456
reserved: [1, 2, 3]
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expectError: true,
			errMsg:      "missing required field: private_key",
		},
		{
			name: "missing public_key",
			input: `private_key: sk_test_123
reserved: [1, 2, 3]
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expectError: true,
			errMsg:      "missing required field: public_key",
		},
		{
			name: "missing reserved",
			input: `private_key: sk_test_123
public_key: pk_test_456
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expectError: true,
			errMsg:      "missing required field: reserved",
		},
		{
			name: "missing v4",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [1, 2, 3]
v6: 2001:db8::1
endpoint: https://example.com`,
			expectError: true,
			errMsg:      "missing required field: v4",
		},
		{
			name: "missing v6",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [1, 2, 3]
v4: 1.2.3.4
endpoint: https://example.com`,
			expectError: true,
			errMsg:      "missing required field: v6",
		},
		{
			name: "missing endpoint",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [1, 2, 3]
v4: 1.2.3.4
v6: 2001:db8::1`,
			expectError: true,
			errMsg:      "missing required field: endpoint",
		},
		{
			name: "malformed reserved",
			input: `private_key: sk_test_123
public_key: pk_test_456
reserved: [1, two, 3]
v4: 1.2.3.4
v6: 2001:db8::1
endpoint: https://example.com`,
			expected: CFCreds{
				SecretKey: "sk_test_123",
				PublicKey: "pk_test_456",
				Reserved:  []int{1, 0, 3}, // 0 for non-numeric value
				V4:        "1.2.3.4",
				V6:        "2001:db8::1",
				Endpoint:  "https://example.com",
			},
			expectError: true,
			errMsg:      "missing required field: reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parseCFCreds(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !errors.Is(err, errors.New(tt.errMsg)) && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if actual.SecretKey != tt.expected.SecretKey {
				t.Errorf("SecretKey mismatch: expected %q, got %q", tt.expected.SecretKey, actual.SecretKey)
			}

			if actual.PublicKey != tt.expected.PublicKey {
				t.Errorf("PublicKey mismatch: expected %q, got %q", tt.expected.PublicKey, actual.PublicKey)
			}

			if len(actual.Reserved) != len(tt.expected.Reserved) {
				t.Errorf("Reserved length mismatch: expected %d, got %d", len(tt.expected.Reserved), len(actual.Reserved))
			} else {
				for i := range actual.Reserved {
					if actual.Reserved[i] != tt.expected.Reserved[i] {
						t.Errorf("Reserved[%d] mismatch: expected %d, got %d", i, tt.expected.Reserved[i], actual.Reserved[i])
					}
				}
			}

			if actual.V4 != tt.expected.V4 {
				t.Errorf("V4 mismatch: expected %q, got %q", tt.expected.V4, actual.V4)
			}

			if actual.V6 != tt.expected.V6 {
				t.Errorf("V6 mismatch: expected %q, got %q", tt.expected.V6, actual.V6)
			}

			if actual.Endpoint != tt.expected.Endpoint {
				t.Errorf("Endpoint mismatch: expected %q, got %q", tt.expected.Endpoint, actual.Endpoint)
			}
		})
	}
}
