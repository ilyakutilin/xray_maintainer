package utils

import (
	"net"
	"time"
)

func WaitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return &net.OpError{Op: "dial", Net: "tcp", Addr: nil, Err: timeoutErr("timed out waiting for port")}
}

type timeoutErr string

func (e timeoutErr) Error() string { return string(e) }
