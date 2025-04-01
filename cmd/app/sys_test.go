package main

import (
	"os"
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
