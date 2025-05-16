package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeReadCloser struct {
	*bytes.Buffer
}

func (f *fakeReadCloser) Close() error { return nil }

func TestWatchXrayStartup(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectReady bool
	}{
		{
			name:        "Xray started line triggers ready",
			input:       "2025/05/16 15:44:52 [Warning] core: Xray 25.4.30 started\n",
			expectReady: true,
		},
		{
			name:        "Failed to start triggers ready",
			input:       "Failed to start: something bad happened\n",
			expectReady: true,
		},
		{
			name:        "No trigger line does not close ready",
			input:       "Some other log line\n",
			expectReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &fakeReadCloser{Buffer: bytes.NewBufferString(tt.input)}
			ready := make(chan struct{})

			go watchXrayStartup(stdout, ready)

			select {
			case <-ready:
				if !tt.expectReady {
					t.Errorf("ready channel closed unexpectedly")
				}
			case <-time.After(200 * time.Millisecond):
				if tt.expectReady {
					t.Errorf("ready channel not closed as expected")
				}
			}
		})
	}
}

func TestWaitForXrayReady_CtxTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ready := make(chan struct{}) // never closed

	err := waitForXrayReady(ctx, ready, 12345)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}
