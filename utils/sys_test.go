package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"testing"
	"time"
)

func TestCheckSudo(t *testing.T) {
	err := CheckSudo()

	if os.Geteuid() == 0 {
		// Running as root - should return nil
		if err != nil {
			t.Errorf("Expected nil error when running as root, got %v", err)
		}
	} else {
		// Not running as root - should return error
		if err == nil {
			t.Error("Expected error when not running as root, got nil")
		}
		expectedErr := "this application requires sudo/root privileges"
		if err.Error() != expectedErr {
			t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
		}
	}
}

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmdStr      string
		timeout     time.Duration
		wantOutput  string
		wantError   bool
		errorSubstr string
	}{
		{
			name:       "successful command",
			cmdStr:     "echo hello world",
			timeout:    2 * time.Second,
			wantOutput: "hello world\n",
			wantError:  false,
		},
		{
			name:        "command with error",
			cmdStr:      "ls /nonexistentdirectory",
			timeout:     2 * time.Second,
			wantError:   true,
			errorSubstr: "command execution failed",
		},
		{
			name:       "empty command",
			cmdStr:     "",
			timeout:    2 * time.Second,
			wantOutput: "",
			wantError:  false,
		},
		{
			name:       "command with spaces",
			cmdStr:     " echo  'test  spaces' ",
			timeout:    2 * time.Second,
			wantOutput: "test  spaces\n",
			wantError:  false,
		},
		{
			name:       "whitespace-only command",
			cmdStr:     "   ",
			timeout:    2 * time.Second,
			wantOutput: "",
			wantError:  false,
		},
		{
			name:        "invalid command syntax",
			cmdStr:      "echo 'unclosed quote",
			timeout:     2 * time.Second,
			wantError:   true,
			errorSubstr: "command execution failed",
		},
		{
			name:        "command times out",
			cmdStr:      "sleep 5",
			timeout:     1 * time.Second,
			wantError:   true,
			errorSubstr: "command timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			output, err := ExecuteCommand(ctx, tt.cmdStr)

			if (err != nil) != tt.wantError {
				t.Errorf("ExecuteCommand() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// For successful cases, check the output
				if output != tt.wantOutput {
					t.Errorf("ExecuteCommand() = %q, want %q", output, tt.wantOutput)
				}
			} else {
				// For error cases, check if the error contains expected substring
				if err != nil && tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("ExecuteCommand() error = %v, want containing %q", err, tt.errorSubstr)
				}

				// Also verify that we get stderr output when there's an error
				if output == "" && tt.name != "command times out" {
					t.Error("ExecuteCommand() returned empty stderr output for failed command")
				}
			}
		})
	}

	t.Run("nonexistent command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := ExecuteCommand(ctx, "nonexistentcommand123")
		if err == nil {
			t.Error("expected error for nonexistent command, got nil")
		}

		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Errorf("expected exec.ExitError, got %T", err)
		}
	})
}

func TestRestartService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tests := []struct {
		name        string
		serviceName string
		mockExec    func(context.Context, string) (string, error)
		wantErr     bool
	}{
		{
			name:        "successful restart",
			serviceName: "nginx",
			mockExec: func(ctx context.Context, cmd string) (string, error) {
				if cmd != "sudo systemctl restart nginx" {
					return "", fmt.Errorf("unexpected command")
				}
				return "", nil
			},
			wantErr: false,
		},
		{
			name:        "failed restart",
			serviceName: "mysql",
			mockExec: func(ctx context.Context, cmd string) (string, error) {
				return "", fmt.Errorf("permission denied")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RestartService(ctx, tt.serviceName, tt.mockExec)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestartService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckServiceStatus(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		mockResponse string
		mockError    error
		expected     bool
		expectErr    bool
	}{
		{
			name:         "active service",
			serviceName:  "nginx",
			mockResponse: "active",
			mockError:    nil,
			expected:     true,
			expectErr:    false,
		},
		{
			name:         "inactive service",
			serviceName:  "mysql",
			mockResponse: "inactive",
			mockError:    nil,
			expected:     false,
			expectErr:    false,
		},
		{
			name:         "service not found",
			serviceName:  "nonexistent",
			mockResponse: "",
			mockError:    fmt.Errorf("Unit nonexistent.service not found"),
			expected:     false,
			expectErr:    true,
		},
		{
			name:         "command execution error",
			serviceName:  "postgres",
			mockResponse: "",
			mockError:    fmt.Errorf("permission denied"),
			expected:     false,
			expectErr:    true,
		},
		{
			name:         "unexpected output",
			serviceName:  "redis",
			mockResponse: "unknown-state",
			mockError:    nil,
			expected:     false,
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			mockExecutor := func(ctx context.Context, cmd string) (string, error) {
				expectedCmd := fmt.Sprintf("systemctl is-active %s", tt.serviceName)
				if cmd != expectedCmd {
					t.Errorf("expected command: %q, got: %q", expectedCmd, cmd)
				}
				return tt.mockResponse, tt.mockError
			}

			active, err := CheckServiceStatus(ctx, tt.serviceName, mockExecutor)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if active != tt.expected {
				t.Errorf("expected active: %v, got: %v", tt.expected, active)
			}
		})
	}
}

func TestCheckServiceStatusWithDefaultExecutor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Save original default executor
	originalExecutor := defaultExecutor
	defer func() { defaultExecutor = originalExecutor }()

	// Set up mock default executor
	defaultExecutor = func(ctx context.Context, cmd string) (string, error) {
		return "active", nil
	}

	active, err := CheckServiceStatus(ctx, "nginx", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !active {
		t.Error("expected service to be active")
	}
}

func TestCheckOperability(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		restartErr     error
		isActive       bool
		checkStatusErr error
		expectedErr    error
	}{
		{
			name:        "successful operation",
			serviceName: "nginx",
			restartErr:  nil,
			isActive:    true,
			expectedErr: nil,
		},
		{
			name:        "restart failure",
			serviceName: "mysql",
			restartErr:  errors.New("permission denied"),
			expectedErr: errors.New("permission denied"),
		},
		{
			name:           "status check failure",
			serviceName:    "redis",
			restartErr:     nil,
			checkStatusErr: errors.New("service not found"),
			expectedErr:    errors.New("service not found"),
		},
		{
			name:        "service not active after restart",
			serviceName: "postgres",
			restartErr:  nil,
			isActive:    false,
			expectedErr: fmt.Errorf("postgres service is not active"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			mockExecutor := func(ctx context.Context, cmd string) (string, error) {
				if strings.Contains(cmd, "sudo systemctl restart") {
					if tt.restartErr != nil {
						return "", tt.restartErr
					}
					return "", nil
				}
				if strings.Contains(cmd, "systemctl is-active") {
					if tt.checkStatusErr != nil {
						return "", tt.checkStatusErr
					}
					if tt.isActive {
						return "active", nil
					}
					return "inactive", nil
				}
				return "", nil
			}

			err := CheckOperability(ctx, tt.serviceName, mockExecutor)

			// Test error conditions
			if tt.expectedErr == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.expectedErr != nil && err == nil {
				t.Errorf("expected error %v, got nil", tt.expectedErr)
			}
			if tt.expectedErr != nil && err != nil && tt.expectedErr.Error() != err.Error() {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestCheckOperabilityIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	currentUser, err := user.Current()
	if err != nil || currentUser.Uid != "0" {
		t.Skip("Skipping integration test - requires root privileges")
	}

	// Test with a real service that should exist on most systems
	serviceName := "cron"
	err = CheckOperability(ctx, serviceName, nil)
	if err != nil {
		t.Errorf("CheckOperability failed: %v", err)
	}
}
