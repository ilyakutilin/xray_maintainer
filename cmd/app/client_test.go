package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

func TestGetClientConfig(t *testing.T) {
	serverConfigGenerator := func(clientID string) ServerConfig {
		return ServerConfig{
			Log: Log{
				Loglevel: "error",
			},
			Inbounds: []SrvInbound{
				{
					Protocol: "vless",
					Tag:      "reality-in",
					Port:     443,
					Listen:   "123.123.123.123",
					Sniffing: SrvInbSniffing{
						Enabled:      true,
						DestOverride: []string{"http", "tls", "quic"},
					},
					Settings: SrvInbSettings{
						Clients: &[]SrvInbSettingsClient{
							{
								ID:    clientID,
								Email: "user1",
								Flow:  "xtls-rprx-vision",
							},
						},
						Decryption: "none",
					},
					StreamSettings: &SrvInbStreamSettings{
						Network:  "tcp",
						Security: "reality",
						RealitySettings: &SrvInbStreamRealitySettings{
							Dest:        "server.com:443",
							Xver:        0,
							ServerNames: []string{"server.com"},
							PrivateKey:  "private_key",
							ShortIds:    []string{""},
						},
					},
				},
			},
		}
	}

	validClientID := "a81bc81b-dead-4e5d-abff-90865d1e13b1"
	tests := []struct {
		name        string
		protocol    string
		clientID    string
		nilExpected bool
		panicMsg    string
	}{
		{
			name:     "success",
			protocol: "vless",
			clientID: validClientID,
		},
		{
			name:     "no required protocol in server inbounds",
			protocol: "required_protocol",
			clientID: validClientID,
			panicMsg: "protocol required_protocol has not been found",
		},
		{
			name:        "no client ID in the server configuration",
			protocol:    "vless",
			clientID:    "",
			nilExpected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConfig := serverConfigGenerator(tt.clientID)

			xrayClient := XrayClient{
				ServerProtocol: tt.protocol,
				Port:           23456,
				PublicKey:      "public_key",
			}

			if tt.panicMsg != "" {
				utils.AssertPanics(t, func() {
					_ = getClientConfig(&xrayClient, &serverConfig)
				}, tt.panicMsg)
				return
			}

			utils.AssertDoesNotPanic(t, func() {
				_ = getClientConfig(&xrayClient, &serverConfig)
			})

			cc := getClientConfig(&xrayClient, &serverConfig)

			if tt.nilExpected {
				fmt.Printf("cc: %v\n", cc)
				if cc != nil {
					t.Errorf("Expected nil, got %v", cc)
				}
				return
			}

			utils.AssertCorrectString(t, "warning", cc.Log.Loglevel)
			utils.AssertCorrectInt(t, 23456, cc.Inbounds[0].Port)
			utils.AssertCorrectString(t, "http", cc.Inbounds[0].Protocol)
			utils.AssertCorrectString(t, "vless", cc.Outbounds[0].Protocol)
			utils.AssertCorrectString(t, "vless", cc.Outbounds[0].Tag)
			utils.AssertCorrectString(t, "123.123.123.123", cc.Outbounds[0].Settings.VNext[0].Address)
			utils.AssertCorrectInt(t, 443, cc.Outbounds[0].Settings.VNext[0].Port)
			utils.AssertCorrectString(t, tt.clientID, cc.Outbounds[0].Settings.VNext[0].Users[0].Id)
			utils.AssertCorrectString(t, "xtls-rprx-vision", cc.Outbounds[0].Settings.VNext[0].Users[0].Flow)
			utils.AssertCorrectString(t, "none", cc.Outbounds[0].Settings.VNext[0].Users[0].Encryption)
			utils.AssertCorrectString(t, "tcp", cc.Outbounds[0].StreamSettings.Network)
			utils.AssertCorrectString(t, "reality", cc.Outbounds[0].StreamSettings.Security)
			utils.AssertCorrectString(t, "server.com", cc.Outbounds[0].StreamSettings.RealitySettings.ServerName)
			utils.AssertCorrectString(t, "edge", cc.Outbounds[0].StreamSettings.RealitySettings.Fingerprint)
			utils.AssertCorrectString(t, "", cc.Outbounds[0].StreamSettings.RealitySettings.ShortId)
			utils.AssertCorrectString(t, "public_key", cc.Outbounds[0].StreamSettings.RealitySettings.PublicKey)
			utils.AssertCorrectString(t, "field", cc.Routing.Rules[0].Type)
			utils.AssertCorrectString(t, "vless", cc.Routing.Rules[0].OutboundTag)
			utils.AssertCorrectString(t, "tcp", cc.Routing.Rules[0].Network)
			utils.AssertCorrectString(t, "IPIfNonMatch", cc.Routing.DomainStrategy)
		})
	}
}

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
