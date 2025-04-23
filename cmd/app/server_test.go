package main

import "testing"

func TestLog_Validate(t *testing.T) {
	tests := []struct {
		name     string
		loglevel string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid debug level",
			loglevel: "debug",
			wantErr:  false,
		},
		{
			name:     "valid info level",
			loglevel: "info",
			wantErr:  false,
		},
		{
			name:     "valid warning level",
			loglevel: "warning",
			wantErr:  false,
		},
		{
			name:     "valid error level",
			loglevel: "error",
			wantErr:  false,
		},
		{
			name:     "valid none level",
			loglevel: "none",
			wantErr:  false,
		},
		{
			name:     "empty loglevel",
			loglevel: "",
			wantErr:  true,
			errMsg:   "xray server config must have the logger block with loglevel set",
		},
		{
			name:     "invalid loglevel",
			loglevel: "critical",
			wantErr:  true,
			errMsg:   `xray server config must have the logger block with loglevel set; allowed values: "debug", "info", "warning", "error", "none"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Log{
				Loglevel: tt.loglevel,
			}
			err := l.Validate()

			if tt.wantErr {
				assertErrorContains(t, err, tt.errMsg)
			} else {
				assertNoError(t, err)
			}
		})
	}
}

func TestSrvInbSniffing_Validate(t *testing.T) {
	tests := []struct {
		name        string
		sniffing    SrvInbSniffing
		wantErr     bool
		errContains string
	}{
		{
			name: "valid with http destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"http"},
			},
			wantErr: false,
		},
		{
			name: "valid with multiple destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic"},
			},
			wantErr: false,
		},
		{
			name: "disabled with empty destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      false,
				DestOverride: []string{},
			},
			wantErr: false,
		},
		{
			name: "enabled but empty destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{},
			},
			wantErr:     true,
			errContains: "must have the inbound block with sniffing enabled and destOverride set",
		},
		{
			name: "invalid destOverride value",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"invalid"},
			},
			wantErr:     true,
			errContains: `allowed values: "http", "tls", "quic"`,
		},
		{
			name: "mixed valid and invalid destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"http", "invalid", "tls"},
			},
			wantErr:     true,
			errContains: `allowed values: "http", "tls", "quic"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sniffing.Validate()
			if tt.wantErr {
				if tt.errContains != "" {
					assertErrorContains(t, err, tt.errContains)
				} else {
					assertError(t, err)
				}
			} else {
				assertNoError(t, err)
			}
		})
	}
}

func TestSrvInbSettingsClient_Validate(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000" // Example valid UUID
	invalidUUID := "not-a-uuid"

	tests := []struct {
		name        string
		client      SrvInbSettingsClient
		wantErr     bool
		errContains string
	}{
		{
			name: "valid client with all fields",
			client: SrvInbSettingsClient{
				ID:    validUUID,
				Email: "user@example.com",
				Flow:  "xtls-rprx-vision",
			},
			wantErr: false,
		},
		{
			name: "valid client without flow",
			client: SrvInbSettingsClient{
				ID:    validUUID,
				Email: "user@example.com",
			},
			wantErr: false,
		},
		{
			name: "invalid UUID",
			client: SrvInbSettingsClient{
				ID:    invalidUUID,
				Email: "user@example.com",
			},
			wantErr:     true,
			errContains: "client id is '" + invalidUUID + "' which is not a valid UUID",
		},
		{
			name: "empty email",
			client: SrvInbSettingsClient{
				ID:    validUUID,
				Email: "",
			},
			wantErr:     true,
			errContains: "client email shall not be empty",
		},
		{
			name: "invalid flow",
			client: SrvInbSettingsClient{
				ID:    validUUID,
				Email: "user@example.com",
				Flow:  "invalid-flow",
			},
			wantErr:     true,
			errContains: "client flow is 'invalid-flow' while only xtls-rprx-vision is allowed",
		},
		{
			name: "multiple errors: invalid UUID and empty email",
			client: SrvInbSettingsClient{
				ID:    invalidUUID,
				Email: "",
			},
			wantErr: true,
			// We can't test for both errors since validation stops at first error
			// This tests that at least one error is caught
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.Validate()
			if tt.wantErr {
				if tt.errContains != "" {
					assertErrorContains(t, err, tt.errContains)
				} else {
					assertError(t, err)
				}
			} else {
				assertNoError(t, err)
			}
		})
	}
}

func TestSrvInbound_Validate(t *testing.T) {
	tests := []struct {
		name        string
		inbound     SrvInbound
		wantErr     bool
		errContains string
	}{
		{
			name: "valid vless protocol",
			inbound: SrvInbound{
				Protocol: "vless",
			},
			wantErr: false,
		},
		{
			name: "valid shadowsocks protocol",
			inbound: SrvInbound{
				Protocol: "shadowsocks",
			},
			wantErr: false,
		},
		{
			name: "empty protocol",
			inbound: SrvInbound{
				Protocol: "",
			},
			wantErr:     true,
			errContains: "inbound.protocol cannot be empty",
		},
		{
			name: "unsupported protocol",
			inbound: SrvInbound{
				Protocol: "unsupported",
			},
			wantErr:     true,
			errContains: "only vless and shadowsocks protocols are supported",
		},
		{
			name: "case sensitivity check",
			inbound: SrvInbound{
				Protocol: "VLESS", // assuming case matters
			},
			wantErr:     true,
			errContains: "only vless and shadowsocks protocols are supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.inbound.Validate()
			if tt.wantErr {
				assertErrorContains(t, err, tt.errContains)
			} else {
				assertNoError(t, err)
			}
		})
	}
}
