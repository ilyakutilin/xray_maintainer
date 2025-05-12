package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
			err := ParseJSONFile(tt.filePath, target, tt.strict)
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
				AssertCorrectString(t, "test", target.Name)
				AssertCorrectInt(t, 30, target.Timeout)
				AssertCorrectBool(t, true, target.Valid)
				AssertCorrectString(t, "localhost", target.Config.Host)
				AssertCorrectInt(t, 8080, target.Config.Port)
			}
		})
	}

	t.Run("nil target", func(t *testing.T) {
		type ConfigStruct struct{}
		err := ParseJSONFile(validJSONFile, (*ConfigStruct)(nil), false)
		if err == nil {
			t.Error("parseJSONFile() error = nil, wantErr target must be a non-nil pointer")
		}
	})
}

func TestWriteStructToJSONFile(t *testing.T) {
	type TestParams struct {
		Method   string `json:"method"`
		Password string `json:"password"`
	}

	type TestConfig struct {
		Host   string     `json:"host"`
		Port   int        `json:"port"`
		Params TestParams `json:"params"`
	}

	validFilePath, cleanupFn := CreateTempFile(t)

	tests := []struct {
		name        string
		cfg         TestConfig
		filePath    string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful write",
			cfg: TestConfig{
				Host: "localhost",
				Port: 8080,
				Params: TestParams{
					Method:   "aes-128-gcm",
					Password: "password123",
				},
			},
			filePath: validFilePath,
			wantErr:  false,
		},
		{
			name:     "empty struct",
			cfg:      TestConfig{},
			filePath: validFilePath,
			wantErr:  false,
		},
		{
			name:        "nonexistent path",
			cfg:         TestConfig{},
			filePath:    "/nonexistent/path/config.json",
			wantErr:     true,
			errContains: "error creating file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(cleanupFn)
			err := WriteStructToJSONFile(tt.cfg, tt.filePath)

			if tt.wantErr {
				AssertErrorContains(t, err, tt.errContains)
			} else {
				AssertNoError(t, err)

				// Read the file back to verify the content
				fileContent, err := os.ReadFile(tt.filePath)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}

				var result TestConfig
				err = json.Unmarshal(fileContent, &result)
				if err != nil {
					t.Fatalf("failed to unmarshal JSON: %v", err)
				}

				AssertCorrectString(t, tt.cfg.Host, result.Host)
				AssertCorrectInt(t, tt.cfg.Port, result.Port)
				AssertCorrectString(t, tt.cfg.Params.Method, result.Params.Method)
				AssertCorrectString(t, tt.cfg.Params.Password, result.Params.Password)
			}
		})
	}
}
