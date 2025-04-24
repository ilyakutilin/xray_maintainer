package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
func ExecuteCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// TODO: Implement timeout
	err := cmd.Run()
	if err != nil {
		return stderr.String(), fmt.Errorf("command execution failed: %w", err)
	}

	return out.String(), nil
}

type CommandExecutor func(string) (string, error)

var defaultExecutor CommandExecutor = ExecuteCommand

func RestartService(serviceName string, executor CommandExecutor) error {
	if executor == nil {
		executor = defaultExecutor
	}

	_, err := executor(fmt.Sprintf("sudo systemctl restart %s", serviceName))
	return err
}

func CheckServiceStatus(serviceName string, executor CommandExecutor) (bool, error) {
	if executor == nil {
		executor = defaultExecutor
	}

	output, err := executor(fmt.Sprintf("systemctl is-active %s", serviceName))
	if err != nil {
		return false, err
	}
	return output == "active", nil
}

func CheckOperability(serviceName string, executor CommandExecutor) error {
	err := RestartService(serviceName, executor)
	if err != nil {
		return err
	}
	isActive, err := CheckServiceStatus(serviceName, executor)
	if err != nil {
		return err
	}
	if !isActive {
		return fmt.Errorf("%s service is not active", serviceName)
	}
	return nil
}
