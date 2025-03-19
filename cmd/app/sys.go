package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Checks if the app has sudo privileges
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

func restartService(serviceName string) error {
	_, err := ExecuteCommand(fmt.Sprintf("sudo systemctl restart %s", serviceName))
	return err
}

func checkServiceStatus(serviceName string) (bool, error) {
	output, err := ExecuteCommand(fmt.Sprintf("systemctl is-active %s", serviceName))
	if err != nil {
		return false, err
	}
	return output == "active", nil
}
