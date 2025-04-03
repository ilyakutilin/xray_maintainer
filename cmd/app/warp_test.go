package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestParseJSONFile(t *testing.T) {
	type TestConfig struct {
		Name    string `json:"name"`
		Timeout int    `json:"timeout"`
		Valid   bool   `json:"valid"`
		Config  struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"config"`
	}

	jsonDir := filepath.Join(t.TempDir(), "json")
	if err := os.Mkdir(jsonDir, 0755); err != nil {
		t.Fatalf("error creating JSON directory: %v", err)
	}

	var (
		validJSONFile         = filepath.Join(jsonDir, "valid.json")
		wrongTypeJSONFile     = filepath.Join(jsonDir, "wrong_type.json")
		invalidJSONFile       = filepath.Join(jsonDir, "invalid.json")
		unknownFieldsJSONFile = filepath.Join(jsonDir, "unknown_fields.json")
		nonexistentJSONFile   = filepath.Join(jsonDir, "nonexistent.json")
	)

	const (
		validData         = `{"name": "test", "timeout": 30, "valid": true, "config": {"host": "localhost", "port": 8080}}`
		wrongTypeData     = `{"name": "test", "timeout": 30, "valid": "true", "config": {"host": "localhost", "port": 8080}}`
		invalidData       = `{invalid json}`
		unknownFieldsData = `{"name": "test", "timeout": 30, "valid": true, "config": {"host": "localhost", "port": 8080}, "extra": "field"}`
	)

	jsonFiles := map[string]string{
		validJSONFile:         validData,
		wrongTypeJSONFile:     wrongTypeData,
		invalidJSONFile:       invalidData,
		unknownFieldsJSONFile: unknownFieldsData,
	}

	for file, data := range jsonFiles {
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Errorf("failed to write file: %v", err)
		}
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(jsonDir); err != nil {
			t.Error(err)
		}
	})

	isConfigZero := func(c *TestConfig) bool {
		return c.Name == "" && c.Timeout == 0 && !c.Valid && c.Config.Host == "" && c.Config.Port == 0
	}

	tests := []struct {
		name        string
		filePath    string
		strict      bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid json to struct",
			filePath: validJSONFile,
			strict:   false,
			wantErr:  false,
		},
		{
			name:     "valid json with strict mode",
			filePath: validJSONFile,
			strict:   true,
			wantErr:  false,
		},
		{
			name:        "nonexistent file",
			filePath:    nonexistentJSONFile,
			strict:      false,
			wantErr:     true,
			errContains: "does not exist",
		},
		{
			name:        "invalid json",
			filePath:    invalidJSONFile,
			strict:      false,
			wantErr:     true,
			errContains: "failed to decode JSON",
		},
		{
			name:     "unknown fields in non-strict mode",
			filePath: unknownFieldsJSONFile,
			strict:   false,
			wantErr:  false,
		},
		{
			name:        "unknown fields in strict mode",
			filePath:    unknownFieldsJSONFile,
			strict:      true,
			wantErr:     true,
			errContains: "unknown field",
		},
		{
			name:        "wrong type in one of the JSON fields",
			filePath:    wrongTypeJSONFile,
			strict:      false,
			wantErr:     true,
			errContains: "cannot unmarshal string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := &TestConfig{}
			err := parseJSONFile(tt.filePath, target, tt.strict)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseJSONFile() error = %v, should contain %q", err, tt.errContains)
				}
			}

			if !tt.wantErr {
				if isConfigZero(target) {
					t.Error("parseJSONFile() target was not modified")
				}
				assertCorrectString(t, "test", target.Name)
				assertCorrectInt(t, 30, target.Timeout)
				assertCorrectBool(t, true, target.Valid)
				assertCorrectString(t, "localhost", target.Config.Host)
				assertCorrectInt(t, 8080, target.Config.Port)
			}
		})
	}

	t.Run("nil target", func(t *testing.T) {
		err := parseJSONFile(validJSONFile, (*Config)(nil), false)
		if err == nil {
			t.Error("parseJSONFile() error = nil, wantErr target must be a non-nil pointer")
		}
	})
}

func TestGetClientConfig(t *testing.T) {
	serverConfigJson := `{
  "log": {
    "loglevel": "error"
  },
  "inbounds": [
    {
	  "port": 12345,
	  "protocol": "shadowsocks",
	  "settings": {
	    "method": "testmethod",
	    "password": "%s",
	    "network": "tcp,udp"
	  }
    }
  ]
}`

	tests := []struct {
		name     string
		protocol string
		password string
		panicMsg string
	}{
		{
			name:     "success",
			protocol: "shadowsocks",
			password: "testpassword",
		},
		{
			name:     "no required protocol in server inbounds",
			protocol: "required_protocol",
			panicMsg: "protocol required_protocol has not been found",
		},
		{
			name:     "no credentials in the server inbound",
			protocol: "shadowsocks",
			password: "",
			panicMsg: "did not provide the required credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmtServerConfigJson := fmt.Sprintf(serverConfigJson, tt.password)

			testDir := t.TempDir()

			t.Cleanup(func() {
				if err := os.RemoveAll(testDir); err != nil {
					t.Error(err)
				}
			})

			serverConfigFile := filepath.Join(testDir, "config.json")

			if err := os.WriteFile(serverConfigFile, []byte(fmtServerConfigJson), 0600); err != nil {
				t.Fatalf("failed to write server config file: %v", err)
			}

			var xrayServerConfig ServerConfig
			// By this point parseJSONFile should have already been tested
			if err := parseJSONFile(serverConfigFile, &xrayServerConfig, true); err != nil {
				t.Fatalf("failed to parse server config file: %v", err)
			}

			warpConfig := Warp{
				xrayProtocol:   tt.protocol,
				xrayClientPort: 23456,
			}

			if tt.panicMsg != "" {
				assertPanics(t, func() {
					_ = getClientConfig(&warpConfig, &xrayServerConfig)
				}, tt.panicMsg)
			} else {
				assertDoesNotPanic(t, func() {
					_ = getClientConfig(&warpConfig, &xrayServerConfig)
				})
				clientConfig := getClientConfig(&warpConfig, &xrayServerConfig)

				assertCorrectInt(t, 23456, clientConfig.Inbounds[0].Port)
				assertCorrectString(t, "http", clientConfig.Inbounds[0].Protocol)
				assertCorrectString(t, tt.protocol, clientConfig.Outbounds[0].Protocol)
				assertCorrectString(t, tt.protocol, clientConfig.Outbounds[0].Tag)
				assertCorrectInt(t, 12345, clientConfig.Outbounds[0].Settings.Servers[0].Port)
				assertCorrectString(t, "testmethod", clientConfig.Outbounds[0].Settings.Servers[0].Method)
				assertCorrectString(t, tt.password, clientConfig.Outbounds[0].Settings.Servers[0].Password)
				assertCorrectString(t, "tcp,udp", clientConfig.Routing.Rules[0].Network)
				assertCorrectString(t, "error", clientConfig.Log.Loglevel)
			}
		})
	}
}
