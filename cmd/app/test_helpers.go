package main

import (
	"math/rand"
	"os"
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

func createTempFile(t testing.TB) (string, func()) {
	t.Helper()
	tempFile, err := os.CreateTemp("", "testfile_"+randomString(8))
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tempFile.Name(), func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}
}
