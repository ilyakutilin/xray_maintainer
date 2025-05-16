package utils

import (
	"net"
	"testing"
	"time"
)

func TestWaitForPort_OpenPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	t.Logf("testing WaitForPort with address: %s", addr)

	err = WaitForPort(addr, 1*time.Second)
	if err != nil {
		t.Errorf("expected port to be available, got error: %v", err)
	}
}

func TestWaitForPort_ClosedPort(t *testing.T) {
	// Pick an arbitrary closed port. Port 1 is usually closed on most systems.
	addr := "127.0.0.1:1"

	err := WaitForPort(addr, 200*time.Millisecond)
	if err == nil {
		t.Errorf("expected error due to port being closed, got nil")
	}
}
