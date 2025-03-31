package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func assertCorrectString(t testing.TB, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func assertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("Wanted an error but didn't get one")
	}
}

func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Wanted no error but got: %v", err)
	}
}

func createTempFile(t testing.TB) (string, func()) {
	t.Helper()
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tempFile.Name(), func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}
}

func TestFileExists(t *testing.T) {
	tempFile, cleanup := createTempFile(t)
	t.Cleanup(cleanup)

	// Test when file exists
	if !fileExists(tempFile) {
		t.Errorf("fileExists shall return true for existing file")
	}

	// Test when file does not exist
	nonExistentPath := tempFile + "_nonexistent"
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
			assertCorrectString(t, test.expected, got)
		})
	}
}

func TestGetStoredReleaseTag(t *testing.T) {
	// Versions file does not exist
	t.Run("Versions file does not exist", func(t *testing.T) {
		tag, err := getStoredReleaseTag("testfile", "doesnotexist.json")
		assertCorrectString(t, "", tag)
		assertNoError(t, err)
	})

	versionsFile, cleanup := createTempFile(t)
	t.Cleanup(cleanup)

	// https://stackoverflow.com/questions/75489773/why-do-i-get-second-argument-to-errors-as-should-not-be-error-build-error-in
	type Error error

	var tests = []struct {
		name           string
		fileContents   []byte
		expectedReturn string
		expectedError  Error
	}{
		{
			name:           "Versions file exists and has correct structure",
			fileContents:   []byte(`{"testfile": "1.2.3"}`),
			expectedReturn: "1.2.3",
			expectedError:  nil,
		},
		{
			name:           "Versions file exists but is empty",
			fileContents:   []byte("{}"),
			expectedReturn: "",
			expectedError:  nil,
		},
		{
			name:           "Versions file exists but is malformed",
			fileContents:   []byte(`{"testfile":`),
			expectedReturn: "",
			expectedError:  &json.SyntaxError{},
		},
		{
			name:           "Versions file has wrong type of value",
			fileContents:   []byte(`{"testfile": 1}`),
			expectedReturn: "",
			expectedError:  &json.UnmarshalTypeError{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := os.WriteFile(versionsFile, test.fileContents, os.ModePerm)
			if err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}
			tag, err := getStoredReleaseTag("testfile", versionsFile)
			assertCorrectString(t, test.expectedReturn, tag)
			if test.expectedError == nil {
				assertNoError(t, err)
			} else {
				if !errors.As(err, &test.expectedError) {
					t.Error("Expected error does not match actual error")
				}
			}
		})
	}

}
