package main

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()
	t.Cleanup(func() {
		os.Remove(tempFilePath)
	})

	// Test when file exists
	if !fileExists(tempFilePath) {
		t.Errorf("fileExists shall return true for existing file")
	}

	// Test when file does not exist
	nonExistentPath := tempFilePath + "_nonexistent"
	if fileExists(nonExistentPath) {
		t.Errorf("fileExists shall return false for non-existing file")
	}
}

func TestExpandPath(t *testing.T) {
	usr, err := user.Current()
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	workingDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	var tests = []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test relative path",
			input:    "./testfile",
			expected: filepath.Join(workingDir, "testfile"),
		},
		{
			name:     "Test absolute path",
			input:    "/testfile",
			expected: "/testfile",
		},
		{
			name:     "Test tilde expansion",
			input:    "~/testfile",
			expected: filepath.Join(usr.HomeDir, "testfile"),
		},
		{
			name:     "Test redundant elements",
			input:    "./test/../testfile",
			expected: filepath.Join(workingDir, "testfile"),
		},
		{
			name:     "Test empty path",
			input:    "",
			expected: workingDir,
		},
		{
			name:     "Test invalid path",
			input:    "../testfile",
			expected: filepath.Join(filepath.Dir(workingDir), "testfile"),
		},
		{
			name:     "Test current directory",
			input:    ".",
			expected: workingDir,
		},
		{
			name:     "Test parent directory",
			input:    "..",
			expected: filepath.Dir(workingDir),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := expandPath(test.input)
			if err != nil {
				t.Errorf("expandPath returned error: %v", err)
			}
			assertCorrectMessage(t, test.expected, got)
		})
	}
}

func assertCorrectMessage(t testing.TB, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}
