package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

func TestCheckPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "check_permissions_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		serviceName string
		workDir     string
		mockOutput  string
		mockError   error
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful permission check",
			serviceName: "test-service",
			workDir:     tempDir,
			mockOutput:  `(root) NOPASSWD: sudo systemctl restart test-service`,
			mockError:   nil,
			wantErr:     false,
		},
		{
			name:        "workdir permission failure",
			serviceName: "test-service",
			workDir:     "/nonexistent/path",
			mockOutput:  `(root) NOPASSWD: sudo systemctl restart test-service`,
			mockError:   nil,
			wantErr:     true,
			errContains: "permission check failed",
		},
		{
			name:        "sudoers check failure",
			serviceName: "test-service",
			workDir:     tempDir,
			mockOutput:  "",
			mockError:   errors.New("sudo command failed"),
			wantErr:     true,
			errContains: "permission check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockExecutor := func(ctx context.Context, cmd string) (string, error) {
				return tt.mockOutput, tt.mockError
			}
			err := CheckPermissions(ctx, tt.serviceName, tt.workDir, mockExecutor)

			if tt.wantErr {
				utils.AssertErrorContains(t, err, tt.errContains)
			} else {
				utils.AssertNoError(t, err)
			}
		})
	}
}

func TestCheckPermissionsIntegration(t *testing.T) {
	// This test can be run with go test -tags=integration
	// It tests with the real executor against the actual system
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will use the real ExecuteCommand function
	err = CheckPermissions(ctx, "nonexistent-service", tempDir, nil)

	// We expect this to fail because the service likely doesn't exist in sudoers
	// but we can verify the error message structure
	utils.AssertErrorContains(t, err, "permission check failed: please add the 'systemctl "+
		"restart nonexistent-service' command to sudoers file")
}
