package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TODO: Write more specific assert functions to replace inline handling

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
