package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TODO: Write more specific assert functions to replace inline handling

func assertCorrectString(t testing.TB, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func assertCorrectInt(t testing.TB, want, got int) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %d, got %d", want, got)
	}
}

func assertCorrectBool(t testing.TB, want, got bool) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %t, got %t", want, got)
	}
}

func assertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("Wanted an error but didn't get one")
	}
}

func assertErrorContains(t testing.TB, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Errorf("Wanted an error but didn't get one")
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("Expected error to contain %q, got %q", substr, err.Error())
	}
}

func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Wanted no error but got: %v", err)
	}
}

// assertPanics checks if fn() panics, and verifies that the panic message contains
// expected substring(s).
// Usage: assertPanics(t, func() { panic("something very bad") }, "bad", "very")
func assertPanics(t *testing.T, fn func(), substrings ...string) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic, but none occurred")
		} else {
			panicStr := fmt.Sprint(r)
			for _, substr := range substrings {
				if !strings.Contains(panicStr, substr) {
					t.Errorf("Panic message %q missing expected substring %q", panicStr, substr)
				}
			}
		}
	}()

	fn()
}

// assertDoesNotPanic checks that fn() does not panic.
// Usage: assertDoesNotPanic(t, func() { doSomethingSafe() })
func assertDoesNotPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()

	fn()
}

// Helper function to generate random strings for filenames
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func createTempFilePath(t testing.TB) string {
	t.Helper()
	tempDir := t.TempDir()
	return filepath.Join(tempDir, "temp-file-"+randomString(8)+".tmp")
}

func createTempFile(t testing.TB) (string, func()) {
	t.Helper()
	tempFilePath := createTempFilePath(t)
	f, err := os.Create(tempFilePath)
	if err != nil {
		t.Fatal(err)
	}
	return tempFilePath, func() {
		err := f.Close()
		if err != nil {
			t.Logf("The file %s is already closed", filepath.Base(tempFilePath))
		}
		err = os.Remove(tempFilePath)
		if err != nil {
			t.Logf("There is nothing to remove by path %s", tempFilePath)
		}
	}
}
