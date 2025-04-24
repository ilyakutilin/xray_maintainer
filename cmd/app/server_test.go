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

func TestSrvInbSettings_Validate(t *testing.T) {
	// Create a minimal valid client slice to use in valid cases
	validClients := []SrvInbSettingsClient{{
		ID:    "123e4567-e89b-12d3-a456-426614174000",
		Email: "valid@example.com",
	}}

	tests := []struct {
		name        string
		settings    SrvInbSettings
		wantErr     bool
		errContains string
	}{
		// Clients validation
		{
			name:        "nil clients",
			settings:    SrvInbSettings{Clients: nil},
			wantErr:     true,
			errContains: "client list should not be empty",
		},
		{
			name: "empty clients slice",
			settings: SrvInbSettings{
				Clients:    &[]SrvInbSettingsClient{},
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "client list should not be empty",
		},

		// Decryption validation
		{
			name: "valid decryption 'none'",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr: false,
		},
		{
			name: "empty decryption",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "decryption cannot be left empty",
		},
		{
			name: "invalid decryption",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "invalid",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "decryption is 'invalid' while only 'none' is allowed",
		},

		// Method validation
		{
			name: "valid method 2022-blake3-aes-128-gcm",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Method:     "2022-blake3-aes-128-gcm",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr: false,
		},
		{
			name: "empty method (valid)",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Method:     "invalid-method",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "method is 'invalid-method' while only the following options are allowed",
		},

		// Password validation
		{
			name: "password too short",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "short",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "password is too short",
		},

		// Network validation
		{
			name: "valid network tcp",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "tcp",
			},
			wantErr: false,
		},
		{
			name: "valid network tcp,udp",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "tcp,udp",
			},
			wantErr: false,
		},
		{
			name: "empty network (invalid)",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "",
			},
			wantErr:     true,
			errContains: "network cannot be left empty",
		},
		{
			name: "invalid network",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Password:   "longenoughpassword123",
				Network:    "invalid",
			},
			wantErr:     true,
			errContains: "network is 'invalid' while only the following options are allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
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

func TestSrvInbStreamRealitySettings(t *testing.T) {
	// Helper valid domains
	validDomain := "example.com:443"
	validDomain2 := "sub.example.com:443"
	invalidDomain := "example..com:443"
	invalidPort := "example.com:80"
	noPort := "example.com"
	ipDest := "1.1.1.1:443"

	tests := []struct {
		name        string
		settings    SrvInbStreamRealitySettings
		wantErr     bool
		errContains string
	}{
		// IsDestValid tests
		{
			name: "valid domain dest",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
			},
			wantErr: false,
		},
		{
			name: "valid subdomain dest",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain2,
				ServerNames: []string{"sub.example.com"},
				PrivateKey:  "test-key",
			},
			wantErr: false,
		},
		{
			name: "valid 1.1.1.1 dest",
			settings: SrvInbStreamRealitySettings{
				Dest:        ipDest,
				ServerNames: []string{""},
				PrivateKey:  "test-key",
			},
			wantErr: false,
		},
		{
			name: "invalid domain format",
			settings: SrvInbStreamRealitySettings{
				Dest:        invalidDomain,
				ServerNames: []string{"example..com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "which is not a valid reality dest",
		},
		{
			name: "invalid port",
			settings: SrvInbStreamRealitySettings{
				Dest:        invalidPort,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "which is not a valid reality dest",
		},
		{
			name: "missing port",
			settings: SrvInbStreamRealitySettings{
				Dest:        noPort,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "which is not a valid reality dest",
		},

		// ValidateServerNames tests
		{
			name: "empty serverNames",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "serverNames must have exactly one element",
		},
		{
			name: "multiple serverNames",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"example.com", "extra.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "serverNames must have exactly one element",
		},
		{
			name: "mismatched serverName",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"wrong.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "serverName 'wrong.com' does not match the domain",
		},
		{
			name: "non-empty serverName with 1.1.1.1",
			settings: SrvInbStreamRealitySettings{
				Dest:        ipDest,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "when the dest is '1.1.1.1:443', serverName must be empty",
		},
		{
			name: "serverName with wildcard",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"*.example.com"},
				PrivateKey:  "test-key",
			},
			wantErr:     true,
			errContains: "wildcards are not suppported in serverNames",
		},

		// Other field validations
		{
			name: "invalid xver",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
				Xver:        1,
			},
			wantErr:     true,
			errContains: "xver is '1' while only 0 is supported",
		},
		{
			name: "empty privateKey",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"example.com"},
				PrivateKey:  "",
			},
			wantErr:     true,
			errContains: "privateKey cannot be empty",
		},
		{
			name: "non-empty shortIds",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"example.com"},
				PrivateKey:  "test-key",
				ShortIds:    []string{"123"},
			},
			wantErr:     true,
			errContains: "shortIds must be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
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

func TestIsDestValid(t *testing.T) {
	tests := []struct {
		name   string
		dest   string
		expect bool
	}{
		{"valid domain", "example.com:443", true},
		{"valid subdomain", "sub.example.com:443", true},
		{"valid ip", "1.1.1.1:443", true},
		{"invalid domain", "example..com:443", false},
		{"wrong port", "example.com:80", false},
		{"missing port", "example.com", false},
		{"empty string", "", false},
		{"invalid format", "not a domain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SrvInbStreamRealitySettings{Dest: tt.dest}
			result := s.IsDestValid()
			assertCorrectBool(t, tt.expect, result)
		})
	}
}

func TestValidateServerNames(t *testing.T) {
	tests := []struct {
		name        string
		settings    SrvInbStreamRealitySettings
		wantErr     bool
		errContains string
	}{
		{"valid domain match", SrvInbStreamRealitySettings{
			Dest: "example.com:443", ServerNames: []string{"example.com"},
		}, false, ""},
		{"valid ip empty", SrvInbStreamRealitySettings{
			Dest: "1.1.1.1:443", ServerNames: []string{""},
		}, false, ""},
		{"empty serverNames", SrvInbStreamRealitySettings{
			Dest: "example.com:443", ServerNames: []string{},
		}, true, "exactly one element"},
		{"multiple serverNames", SrvInbStreamRealitySettings{
			Dest: "example.com:443", ServerNames: []string{"a", "b"},
		}, true, "exactly one element"},
		{"mismatched domain", SrvInbStreamRealitySettings{
			Dest: "example.com:443", ServerNames: []string{"wrong.com"},
		}, true, "does not match"},
		{"ip with non-empty", SrvInbStreamRealitySettings{
			Dest: "1.1.1.1:443", ServerNames: []string{"example.com"},
		}, true, "must be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.ValidateServerNames()
			if tt.wantErr {
				assertErrorContains(t, err, tt.errContains)
			} else {
				assertNoError(t, err)
			}
		})
	}
}

func TestSrvInbStreamSettings_Validate(t *testing.T) {
	// Setup valid reality settings to use in tests
	validReality := SrvInbStreamRealitySettings{
		Dest:        "example.com:443",
		ServerNames: []string{"example.com"},
		PrivateKey:  "valid-private-key",
	}

	// Setup invalid reality settings to test error propagation
	invalidReality := SrvInbStreamRealitySettings{
		Dest:        "invalid.example.com:443",
		ServerNames: []string{""},
		PrivateKey:  "",
	}

	tests := []struct {
		name        string
		settings    SrvInbStreamSettings
		wantErr     bool
		errContains string
	}{
		// Valid cases
		{
			name: "valid tcp with reality",
			settings: SrvInbStreamSettings{
				Network:         "tcp",
				Security:        "reality",
				RealitySettings: validReality,
			},
			wantErr: false,
		},
		{
			name: "valid raw with reality",
			settings: SrvInbStreamSettings{
				Network:         "raw",
				Security:        "reality",
				RealitySettings: validReality,
			},
			wantErr: false,
		},

		// Network validation tests
		{
			name: "empty network",
			settings: SrvInbStreamSettings{
				Network:         "",
				Security:        "reality",
				RealitySettings: validReality,
			},
			wantErr:     true,
			errContains: "network cannot be empty",
		},
		{
			name: "invalid network",
			settings: SrvInbStreamSettings{
				Network:         "udp",
				Security:        "reality",
				RealitySettings: validReality,
			},
			wantErr:     true,
			errContains: "network is 'udp' while only 'raw' or 'tcp'",
		},

		// Security validation tests
		{
			name: "empty security",
			settings: SrvInbStreamSettings{
				Network:         "tcp",
				Security:        "",
				RealitySettings: validReality,
			},
			wantErr:     true,
			errContains: "security cannot be empty",
		},
		{
			name: "invalid security",
			settings: SrvInbStreamSettings{
				Network:         "tcp",
				Security:        "tls",
				RealitySettings: validReality,
			},
			wantErr:     true,
			errContains: "only 'reality' security is supported",
		},

		// RealitySettings validation propagation
		{
			name: "invalid reality settings",
			settings: SrvInbStreamSettings{
				Network:         "tcp",
				Security:        "reality",
				RealitySettings: invalidReality,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
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

// func TestSrvInbound_Validate(t *testing.T) {
// 	tests := []struct {
// 		name        string
// 		inbound     SrvInbound
// 		wantErr     bool
// 		errContains string
// 	}{
// 		{
// 			name: "valid vless protocol",
// 			inbound: SrvInbound{
// 				Protocol: "vless",
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "valid shadowsocks protocol",
// 			inbound: SrvInbound{
// 				Protocol: "shadowsocks",
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "empty protocol",
// 			inbound: SrvInbound{
// 				Protocol: "",
// 			},
// 			wantErr:     true,
// 			errContains: "inbound.protocol cannot be empty",
// 		},
// 		{
// 			name: "unsupported protocol",
// 			inbound: SrvInbound{
// 				Protocol: "unsupported",
// 			},
// 			wantErr:     true,
// 			errContains: "only vless and shadowsocks protocols are supported",
// 		},
// 		{
// 			name: "case sensitivity check",
// 			inbound: SrvInbound{
// 				Protocol: "VLESS", // assuming case matters
// 			},
// 			wantErr:     true,
// 			errContains: "only vless and shadowsocks protocols are supported",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := tt.inbound.Validate()
// 			if tt.wantErr {
// 				assertErrorContains(t, err, tt.errContains)
// 			} else {
// 				assertNoError(t, err)
// 			}
// 		})
// 	}
// }

func TestSrvInbound_Validate(t *testing.T) {
	// Helper valid configurations
	validSniffing := SrvInbSniffing{
		Enabled:      true,
		DestOverride: []string{"http"},
	}
	validSettings := SrvInbSettings{
		Clients:    &[]SrvInbSettingsClient{{ID: "123e4567-e89b-12d3-a456-426614174000", Email: "test@example.com"}},
		Decryption: "none",
		Password:   "longenoughpassword123",
		Network:    "tcp",
	}
	validStreamSettings := &SrvInbStreamSettings{
		RealitySettings: SrvInbStreamRealitySettings{
			Dest:        "example.com:443",
			ServerNames: []string{"example.com"},
			PrivateKey:  "valid-key",
		},
	}

	// External IPv4 for testing
	externalIPv4 := "203.0.113.1"

	tests := []struct {
		name        string
		inbound     SrvInbound
		wantErr     bool
		errContains string
	}{
		// Protocol validation
		{
			name: "valid vless",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr: false,
		},
		{
			name: "valid shadowsocks",
			inbound: SrvInbound{
				Protocol: "shadowsocks",
				Tag:      "ss-in",
				Port:     8388,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr: false,
		},
		{
			name: "empty protocol",
			inbound: SrvInbound{
				Protocol: "",
				Tag:      "test",
				Port:     443,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "inbound.protocol cannot be empty",
		},
		{
			name: "invalid protocol",
			inbound: SrvInbound{
				Protocol: "vmess",
				Tag:      "test",
				Port:     443,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "only vless and shadowsocks protocols are supported",
		},

		// Tag validation
		{
			name: "empty tag",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "inbound.tag cannot be empty",
		},

		// Port validation
		{
			name: "vless with wrong port",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     8080,
				Listen:   externalIPv4,
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "inbound.port: vless protocol only supports port 443",
		},

		// Listen IP validation
		{
			name: "vless with internal IP",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   "192.168.1.1",
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "inbound.listen shall be an external IPv4 address",
		},
		{
			name: "vless with empty listen",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   "",
				Sniffing: validSniffing,
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "inbound.listen shall be an external IPv4 address",
		},

		// Nested validation
		{
			name: "invalid sniffing",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: SrvInbSniffing{
					Enabled:      true,
					DestOverride: []string{},
				},
				Settings: validSettings,
			},
			wantErr:     true,
			errContains: "sniffing enabled and destOverride set",
		},
		{
			name: "invalid settings",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: validSniffing,
				Settings: SrvInbSettings{
					Clients:    &[]SrvInbSettingsClient{},
					Decryption: "none",
					Password:   "short",
					Network:    "tcp",
				},
			},
			wantErr:     true,
			errContains: "client list should not be empty",
		},
		{
			name: "invalid stream settings",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: validSniffing,
				Settings: validSettings,
				StreamSettings: &SrvInbStreamSettings{
					Network:         "udp",
					Security:        "reality",
					RealitySettings: validStreamSettings.RealitySettings,
				},
			},
			wantErr:     true,
			errContains: "network is 'udp' while only 'raw' or 'tcp'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.inbound.Validate()
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

func TestSrvOutboundSettingsPeer_Validate(t *testing.T) {
	validEndpoint := "engage.cloudflareclient.com:2408"
	validPublicKey := "valid-public-key"

	tests := []struct {
		name        string
		peer        SrvOutboundSettingsPeer
		wantErr     bool
		errContains string
	}{
		{
			name: "valid peer",
			peer: SrvOutboundSettingsPeer{
				Endpoint:  validEndpoint,
				PublicKey: validPublicKey,
			},
			wantErr: false,
		},
		{
			name: "empty endpoint",
			peer: SrvOutboundSettingsPeer{
				Endpoint:  "",
				PublicKey: validPublicKey,
			},
			wantErr:     true,
			errContains: "endpoint '' is not a valid endpoint",
		},
		{
			name: "empty public key",
			peer: SrvOutboundSettingsPeer{
				Endpoint:  validEndpoint,
				PublicKey: "",
			},
			wantErr:     true,
			errContains: "publicKey cannot be empty",
		},
		{
			name: "invalid endpoint (no port)",
			peer: SrvOutboundSettingsPeer{
				Endpoint:  "engage.cloudflareclient.com",
				PublicKey: validPublicKey,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.peer.Validate()
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

func TestSrvOutbSettings_Validate(t *testing.T) {
	validPeer := SrvOutboundSettingsPeer{"example.com:8080", "valid-public-key"}

	tests := []struct {
		name        string
		settings    SrvOutbSettings
		wantErr     bool
		errContains string
	}{
		// Valid cases
		{
			name: "valid minimal configuration",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1, 2, 3},
				Workers:        1,
				DomainStrategy: "ForceIPv4",
			},
			wantErr: false,
		},
		{
			name: "valid full configuration",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1", "10.0.0.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer, validPeer},
				Mtu:            1500,
				Reserved:       []int{0},
				Workers:        4,
				DomainStrategy: "ForceIP",
			},
			wantErr: false,
		},

		// SecretKey validation
		{
			name: "empty secretKey",
			settings: SrvOutbSettings{
				SecretKey:      "",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1},
				Workers:        1,
				DomainStrategy: "ForceIP",
			},
			wantErr:     true,
			errContains: "outbound.settings.secretKey cannot be empty",
		},

		// Address validation
		{
			name: "empty address array",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1},
				Workers:        1,
				DomainStrategy: "ForceIP",
			},
			wantErr:     true,
			errContains: "outbound.settings.address array cannot be empty",
		},

		// MTU validation
		{
			name: "MTU too small",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1279,
				Reserved:       []int{1},
				Workers:        1,
				DomainStrategy: "ForceIP",
			},
			wantErr:     true,
			errContains: "outbound.settings.mtu must be between 1280 and 1500",
		},

		// Reserved validation
		{
			name: "empty reserved array",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{},
				Workers:        1,
				DomainStrategy: "ForceIP",
			},
			wantErr:     true,
			errContains: "outbound.settings.reserved array cannot be empty",
		},

		// Workers validation
		{
			name: "zero workers",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1},
				Workers:        0,
				DomainStrategy: "ForceIP",
			},
			wantErr:     true,
			errContains: "outbound.settings.workers must be at least 1",
		},

		// DomainStrategy validation
		{
			name: "empty domainStrategy",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1},
				Workers:        1,
				DomainStrategy: "",
			},
			wantErr:     true,
			errContains: "outbound.settings.domainStrategy cannot be empty",
		},
		{
			name: "invalid domainStrategy",
			settings: SrvOutbSettings{
				SecretKey:      "test-key",
				Address:        []string{"192.168.1.1"},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1280,
				Reserved:       []int{1},
				Workers:        1,
				DomainStrategy: "InvalidStrategy",
			},
			wantErr:     true,
			errContains: "outbound.settings.domainStrategy is 'InvalidStrategy' while only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
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

func TestSrvOutbound_Validate(t *testing.T) {
	// Helper valid settings for WireGuard
	validSettings := &SrvOutbSettings{
		SecretKey:      "test-key",
		Address:        []string{"192.168.1.1"},
		Peers:          []SrvOutboundSettingsPeer{{"example.com:8080", "valid-public-key"}},
		Mtu:            1280,
		Reserved:       []int{1, 2, 3},
		Workers:        2,
		DomainStrategy: "ForceIP",
	}

	tests := []struct {
		name        string
		outbound    SrvOutbound
		wantErr     bool
		errContains string
	}{
		// Protocol validation
		{
			name: "valid blackhole",
			outbound: SrvOutbound{
				Protocol: "blackhole",
				Tag:      "blackhole-out",
			},
			wantErr: false,
		},
		{
			name: "valid freedom",
			outbound: SrvOutbound{
				Protocol: "freedom",
				Tag:      "freedom-out",
			},
			wantErr: false,
		},
		{
			name: "valid wireguard with settings",
			outbound: SrvOutbound{
				Protocol: "wireguard",
				Tag:      "wg-out",
				Settings: validSettings,
			},
			wantErr: false,
		},
		{
			name: "empty protocol",
			outbound: SrvOutbound{
				Protocol: "",
				Tag:      "test",
			},
			wantErr:     true,
			errContains: "outbound.protocol cannot be empty",
		},
		{
			name: "invalid protocol",
			outbound: SrvOutbound{
				Protocol: "invalid",
				Tag:      "test",
			},
			wantErr:     true,
			errContains: "invalid outbound.protocol 'invalid'",
		},

		// Tag validation
		{
			name: "empty tag",
			outbound: SrvOutbound{
				Protocol: "freedom",
				Tag:      "",
			},
			wantErr:     true,
			errContains: "outbound.tag cannot be empty",
		},

		// WireGuard special case
		{
			name: "wireguard without settings",
			outbound: SrvOutbound{
				Protocol: "wireguard",
				Tag:      "wg-out",
				Settings: nil,
			},
			wantErr:     true,
			errContains: "outbound.settings cannot be empty for wireguard protocol",
		},

		// Settings validation propagation
		{
			name: "invalid settings",
			outbound: SrvOutbound{
				Protocol: "wireguard",
				Tag:      "wg-out",
				Settings: &SrvOutbSettings{
					SecretKey:      "", // Invalid empty secretKey
					Address:        []string{"192.168.1.1"},
					Peers:          []SrvOutboundSettingsPeer{{}},
					Mtu:            1280,
					Reserved:       []int{1},
					Workers:        1,
					DomainStrategy: "ForceIPv6",
				},
			},
			wantErr:     true,
			errContains: "outbound.settings.secretKey cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.outbound.Validate()
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

func TestSrvRoutingRule_Validate(t *testing.T) {
	tests := []struct {
		name        string
		rule        SrvRoutingRule
		wantErr     bool
		errContains string
	}{
		// Valid cases
		{
			name: "valid minimal field rule",
			rule: SrvRoutingRule{
				Type:        "field",
				OutboundTag: "proxy-out",
			},
			wantErr: false,
		},
		{
			name: "valid rule with optional fields",
			rule: SrvRoutingRule{
				Type:        "field",
				OutboundTag: "block-out",
				Protocol:    "http",
				Domain:      []string{"example.com"},
				IP:          []string{"1.1.1.1"},
			},
			wantErr: false,
		},

		// Type validation
		{
			name: "empty type",
			rule: SrvRoutingRule{
				Type:        "",
				OutboundTag: "test",
			},
			wantErr:     true,
			errContains: "routing.rules.type cannot be empty",
		},
		{
			name: "invalid type",
			rule: SrvRoutingRule{
				Type:        "invalid",
				OutboundTag: "test",
			},
			wantErr:     true,
			errContains: "invalid routing.rules.type 'invalid'",
		},

		// OutboundTag validation
		{
			name: "empty outboundTag",
			rule: SrvRoutingRule{
				Type:        "field",
				OutboundTag: "",
			},
			wantErr:     true,
			errContains: "routing.rules.outboundTag cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
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
