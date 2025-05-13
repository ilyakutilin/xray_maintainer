package utils

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetRequest_Success(t *testing.T) {
	// Setup a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer ts.Close()

	ctx := context.Background()
	body, err := GetRequest(ctx, ts.URL)

	AssertNoError(t, err)

	AssertCorrectString(t, string(body), "test response")
}

func TestGetRequest_Non200Status(t *testing.T) {
	// Setup a test HTTP server that returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := GetRequest(ctx, ts.URL)

	AssertError(t, err)
	AssertErrorContains(t, err, "received bad status code when making http request to")
}

func TestGetRequest_ContextTimeout(t *testing.T) {
	// Setup a test HTTP server that delays response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Longer than our test context timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := GetRequest(ctx, ts.URL)

	AssertError(t, err)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestGetRequest_InvalidURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: "empty URL provided",
		},
		{
			name:     "malformed URL",
			url:      "http://[::1]:namedport",
			expected: "invalid URL format",
		},
		{
			name:     "missing host",
			url:      "http://",
			expected: "missing host in URL",
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetRequest(ctx, tc.url)
			AssertError(t, err)
			AssertErrorContains(t, err, tc.expected)
		})
	}
}

func TestGetRequest_ReadBodyError(t *testing.T) {
	// Setup a test HTTP server that closes the connection immediately after headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Hijack the connection to close it prematurely
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("webserver doesn't support hijacking")
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := GetRequest(ctx, ts.URL)

	AssertError(t, err)
	AssertErrorContains(t, err, "failed to read response body")
}
