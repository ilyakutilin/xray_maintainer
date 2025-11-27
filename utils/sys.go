package utils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

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

func RestartService(
	ctx context.Context, serviceName string, executor CommandExecutor,
) error {
	if executor == nil {
		executor = defaultExecutor
	}

	_, err := executor(ctx, fmt.Sprintf("sudo systemctl restart %s", serviceName))
	return err
}

func CheckServiceStatus(
	ctx context.Context, serviceName string, executor CommandExecutor,
) (bool, error) {
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

func CheckOperability(
	ctx context.Context, serviceName string, executor CommandExecutor,
) error {
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
func checkCommandInSudoers(
	ctx context.Context, cmdStr string, executor CommandExecutor,
) error {
	output, err := executor(ctx, "sudo -l")
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	var found bool

	trimmedTarget := strings.ReplaceAll(
		strings.TrimSpace(cmdStr), "sudo ", "",
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(line, "NOPASSWD") &&
			strings.HasSuffix(trimmedLine, trimmedTarget) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("please add the '%s' command to sudoers file: run "+
			"'sudo visudo' and add the line 'username ALL=(root) NOPASSWD: %s' "+
			"where username is your user name", trimmedTarget, trimmedTarget)
	}

	return nil
}

// CheckPermissions checks the permissions to read / write to the workdir
// and execute the service restart command by the current user
func CheckPermissions(
	ctx context.Context, serviceName string, workDir string, executor CommandExecutor,
) error {
	if executor == nil {
		executor = defaultExecutor
	}

	// TODO: Instead of hardcodig false make it dependable on dryRun
	if err := checkDirPermissions(workDir, false); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	restartCmd := fmt.Sprintf("sudo systemctl restart %s", serviceName)

	if err := checkCommandInSudoers(ctx, restartCmd, executor); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	return nil
}
