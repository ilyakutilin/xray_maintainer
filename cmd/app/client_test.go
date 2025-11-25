package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// func TestGetClientConfig(t *testing.T) {
// 	serverConfigJson := `{
//   "log": {
//     "loglevel": "error"
//   },
//   "inbounds": [
//     {
// 	  "port": 12345,
// 	  "protocol": "shadowsocks",
// 	  "settings": {
// 	    "method": "testmethod",
// 	    "password": "%s",
// 	    "network": "tcp,udp"
// 	  }
//     }
//   ]
// }`

// 	tests := []struct {
// 		name     string
// 		protocol string
// 		password string
// 		panicMsg string
// 	}{
// 		{
// 			name:     "success",
// 			protocol: "shadowsocks",
// 			password: "testpassword",
// 		},
// 		{
// 			name:     "no required protocol in server inbounds",
// 			protocol: "required_protocol",
// 			panicMsg: "protocol required_protocol has not been found",
// 		},
// 		{
// 			name:     "no credentials in the server inbound",
// 			protocol: "shadowsocks",
// 			password: "",
// 			panicMsg: "did not provide the required credentials",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			fmtServerConfigJson := fmt.Sprintf(serverConfigJson, tt.password)

// 			testDir := t.TempDir()

// 			t.Cleanup(func() {
// 				if err := os.RemoveAll(testDir); err != nil {
// 					t.Error(err)
// 				}
// 			})

// 			serverConfigFile := filepath.Join(testDir, "config.json")

// 			if err := os.WriteFile(serverConfigFile, []byte(fmtServerConfigJson), 0600); err != nil {
// 				t.Fatalf("failed to write server config file: %v", err)
// 			}

// 			var xrayServerConfig ServerConfig
// 			// By this point parseJSONFile should have already been tested
// 			if err := utils.ParseJSONFile(serverConfigFile, &xrayServerConfig, true); err != nil {
// 				t.Fatalf("failed to parse server config file: %v", err)
// 			}

// 			xrayClient := XrayClient{
// 				ServerProtocol: tt.protocol,
// 				Port:           23456,
// 			}

// 			xrayServer := XrayServer{
// 				IP: "1.1.1.1",
// 			}

// 			if tt.panicMsg != "" {
// 				utils.AssertPanics(t, func() {
// 					_ = getClientConfig(&xrayClient, &xrayServer, &xrayServerConfig)
// 				}, tt.panicMsg)
// 			} else {
// 				utils.AssertDoesNotPanic(t, func() {
// 					_ = getClientConfig(&xrayClient, &xrayServer, &xrayServerConfig)
// 				})
// 				clientConfig := getClientConfig(&xrayClient, &xrayServer, &xrayServerConfig)

// 				utils.AssertCorrectInt(t, 23456, clientConfig.Inbounds[0].Port)
// 				utils.AssertCorrectString(t, "http", clientConfig.Inbounds[0].Protocol)
// 				utils.AssertCorrectString(t, tt.protocol, clientConfig.Outbounds[0].Protocol)
// 				utils.AssertCorrectString(t, tt.protocol, clientConfig.Outbounds[0].Tag)
// 				utils.AssertCorrectInt(t, 12345, clientConfig.Outbounds[0].Settings.Servers[0].Port)
// 				utils.AssertCorrectString(t, "testmethod", clientConfig.Outbounds[0].Settings.Servers[0].Method)
// 				utils.AssertCorrectString(t, tt.password, clientConfig.Outbounds[0].Settings.Servers[0].Password)
// 				utils.AssertCorrectString(t, "tcp,udp", clientConfig.Routing.Rules[0].Network)
// 			}
// 		})
// 	}
// }

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
