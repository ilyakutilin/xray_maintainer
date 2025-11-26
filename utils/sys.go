package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Checks if the app has sudo privileges
// TODO: CheckSudo() is currently used only in the tests - check implementation!
func CheckSudo() error {
	if os.Geteuid() != 0 {
		return errors.New("this application requires sudo/root privileges")
	}
	return nil
}

// Runs a shell command and returns its output or an error.
func ExecuteCommand(ctx context.Context, cmdStr string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return stderr.String(), fmt.Errorf("command timed out: %w", ctx.Err())
	}
	if err != nil {
		return stderr.String(), fmt.Errorf("command execution failed: %w", err)
	}

	return out.String(), nil
}

type CommandExecutor func(context.Context, string) (string, error)

var defaultExecutor CommandExecutor = ExecuteCommand

func RestartService(ctx context.Context, serviceName string, executor CommandExecutor) error {
	if executor == nil {
		executor = defaultExecutor
	}

	_, err := executor(ctx, fmt.Sprintf("sudo systemctl restart %s", serviceName))
	return err
}

func CheckServiceStatus(ctx context.Context, serviceName string, executor CommandExecutor) (bool, error) {
	if executor == nil {
		executor = defaultExecutor
	}

	for range 5 {
		output, err := executor(ctx, fmt.Sprintf("systemctl is-active %s", serviceName))
		if err != nil {
			return false, err
		}
		if strings.ReplaceAll(output, "\n", "") != "active" {
			time.Sleep(1 * time.Second)
		} else {
			return true, nil
		}
	}
	return false, nil
}

func CheckOperability(ctx context.Context, serviceName string, executor CommandExecutor) error {
	err := RestartService(ctx, serviceName, executor)
	if err != nil {
		return err
	}
	isActive, err := CheckServiceStatus(ctx, serviceName, executor)
	if err != nil {
		return err
	}
	if !isActive {
		return fmt.Errorf("%s service is not active", serviceName)
	}
	return nil
}

// checkCommandInSudoers checks if the command is added to the sudoers file
// and its execution by the current user is allowed without a password
func checkCommandInSudoers(ctx context.Context, cmdStr string) error {
	output, err := ExecuteCommand(ctx, "sudo -l")
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	var found bool

	lineEndsWithCommand := func(line string, targetCommand string) bool {
		trimmedLine := strings.TrimSpace(line)
		trimmedTarget := strings.TrimSpace(targetCommand)

		return strings.HasSuffix(trimmedLine, trimmedTarget)
	}

	for _, line := range lines {
		if strings.Contains(line, "NOPASSWD") && lineEndsWithCommand(line, cmdStr) {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("please add the '%s' command to sudoers file: run "+
			"'sudo visudo' and add the line 'username ALL=(root) NOPASSWD: %s' "+
			"where username is your user name", cmdStr, cmdStr)
	}

	return nil
}

// checkPathPermissions checks if the current user has read and write permissions
// to the given path
func checkPathPermissions(path string) error {
	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.New("path does not exist: " + path)
		}
		return err
	}

	// Check read permission by attempting to read
	if _, err := os.ReadDir(path); err != nil {
		// If directory read fails, try file read
		if _, err := os.ReadFile(path); err != nil {
			return errors.New("no read permission: " + path)
		}
	}

	// Check write permission by attempting to write a temporary file
	tempFile, err := os.CreateTemp(path, "perm_test_")
	if err != nil {
		return errors.New("no write permission: " + path)
	}
	tempFile.Close()
	os.Remove(tempFile.Name())

	return nil
}

// CheckPermissions checks the permissions to read / write to the workdir
// and execute the service restart command by the current user
func CheckPermissions(ctx context.Context, restartCmd string, workDir string) error {
	if err := checkPathPermissions(workDir); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	if err := checkCommandInSudoers(ctx, restartCmd); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	return nil
}
