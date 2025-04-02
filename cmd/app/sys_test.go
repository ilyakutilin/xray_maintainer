package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCheckSudo(t *testing.T) {
	err := checkSudo()

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
		wantOutput  string
		wantError   bool
		errorSubstr string
	}{
		{
			name:       "successful command",
			cmdStr:     "echo hello world",
			wantOutput: "hello world\n",
			wantError:  false,
		},
		{
			name:        "command with error",
			cmdStr:      "ls /nonexistentdirectory",
			wantError:   true,
			errorSubstr: "command execution failed",
		},
		{
			name:       "empty command",
			cmdStr:     "",
			wantOutput: "",
			wantError:  false,
		},
		{
			name:       "command with spaces",
			cmdStr:     " echo  'test  spaces' ",
			wantOutput: "test  spaces\n",
			wantError:  false,
		},
		{
			name:       "whitespace-only command",
			cmdStr:     "   ",
			wantOutput: "",
			wantError:  false,
		},
		{
			name:        "invalid command syntax",
			cmdStr:      "echo 'unclosed quote",
			wantError:   true,
			errorSubstr: "command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(tt.cmdStr)

			if (err != nil) != tt.wantError {
				t.Errorf("executeCommand() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// For successful cases, check the output
				if output != tt.wantOutput {
					t.Errorf("executeCommand() = %q, want %q", output, tt.wantOutput)
				}
			} else {
				// For error cases, check if the error contains expected substring
				if err != nil && tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("executeCommand() error = %v, want containing %q", err, tt.errorSubstr)
				}

				// Also verify that we get stderr output when there's an error
				if output == "" {
					t.Error("executeCommand() returned empty stderr output for failed command")
				}
			}
		})
	}

	t.Run("nonexistent command", func(t *testing.T) {
		_, err := executeCommand("nonexistentcommand123")
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
	tests := []struct {
		name        string
		serviceName string
		mockExec    func(string) (string, error)
		wantErr     bool
	}{
		{
			name:        "successful restart",
			serviceName: "nginx",
			mockExec: func(cmd string) (string, error) {
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
			mockExec: func(cmd string) (string, error) {
				return "", fmt.Errorf("permission denied")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := restartService(tt.serviceName, tt.mockExec)
			if (err != nil) != tt.wantErr {
				t.Errorf("restartService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
