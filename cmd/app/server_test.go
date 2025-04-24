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
			name: "disabled",
			sniffing: SrvInbSniffing{
				Enabled: false,
			},
			wantErr:     true,
			errContains: "inbound.sniffing must be enabled",
		},
		{
			name: "enabled but empty destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{},
			},
			wantErr:     true,
			errContains: "inbound.sniffing.destOverride shall be set",
		},
		{
			name: "invalid destOverride value",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"invalid"},
			},
			wantErr:     true,
			errContains: "wrong value for inbound.sniffing.destOverride: 'invalid'. ",
		},
		{
			name: "mixed valid and invalid destOverride",
			sniffing: SrvInbSniffing{
				Enabled:      true,
				DestOverride: []string{"http", "invalid", "tls"},
			},
			wantErr:     true,
			errContains: "wrong value for inbound.sniffing.destOverride: 'invalid'. ",
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
			errContains: "inbound.settings.client.id is '" + invalidUUID + "' which is not a valid UUID",
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
			errContains: "inbound.settings.client.flow is 'invalid-flow' while only xtls-rprx-vision is allowed",
		},
		{
			name: "multiple errors: invalid UUID and empty email",
			client: SrvInbSettingsClient{
				ID:    invalidUUID,
				Email: "",
			},
			wantErr:     true,
			errContains: "\n",
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
		// Empty Clients are allowed
		{
			name: "nil clients",
			settings: SrvInbSettings{
				Clients:    nil,
				Decryption: "none",
			},
			wantErr: false,
		},
		{
			name: "empty clients slice",
			settings: SrvInbSettings{
				Clients:    &[]SrvInbSettingsClient{},
				Decryption: "none",
			},
			wantErr: false,
		},

		// Mutual exclusivity validation
		{
			name: "vless, no shadowsocks",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
			},
			wantErr: false,
		},
		{
			name: "shadowsocks, no vless",
			settings: SrvInbSettings{
				Method:   "2022-blake3-aes-128-gcm",
				Password: "longenoughpassword123",
				Network:  "tcp",
			},
			wantErr: false,
		},
		{
			name: "mix of vless and shadowsocks",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "none",
				Network:    "tcp",
			},
			wantErr:     true,
			errContains: "cannot set inbound.settings.decryption together",
		},
		{
			name:        "empty settings",
			settings:    SrvInbSettings{},
			wantErr:     true,
			errContains: "either inbound.settings.decryption or",
		},
		{
			name: "incomplete shadowsocks",
			settings: SrvInbSettings{
				Method: "2022-blake3-aes-128-gcm",
			},
			wantErr:     true,
			errContains: "shall all be set if at least one of them is set",
		},

		// Decryption validation
		{
			name: "invalid decryption",
			settings: SrvInbSettings{
				Clients:    &validClients,
				Decryption: "invalid",
			},
			wantErr:     true,
			errContains: "inbound.settings.decryption is 'invalid' while only 'none' is allowed",
		},

		// Method validation
		{
			name: "valid method",
			settings: SrvInbSettings{
				Method:   "2022-blake3-aes-128-gcm",
				Password: "longenoughpassword123",
				Network:  "tcp,udp",
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			settings: SrvInbSettings{
				Method:   "invalid-method",
				Password: "longenoughpassword123",
				Network:  "tcp",
			},
			wantErr:     true,
			errContains: "inbound.settings.method is 'invalid-method' while only the following options are allowed",
		},

		// Password validation
		{
			name: "password too short",
			settings: SrvInbSettings{
				Method:   "2022-blake3-aes-128-gcm",
				Password: "short",
				Network:  "tcp",
			},
			wantErr:     true,
			errContains: "inbound.settings.password is too short",
		},

		// Network validation
		{
			name: "invalid network",
			settings: SrvInbSettings{
				Method:   "2022-blake3-aes-128-gcm",
				Password: "longenoughpassword123",
				Network:  "invalid",
			},
			wantErr:     true,
			errContains: "network is 'invalid' while only the following options are allowed",
		},

		// Multiple errors
		{
			name: "multiple errors",
			settings: SrvInbSettings{
				Method:   "invalid-method",
				Password: "short",
				Network:  "invalid",
			},
			wantErr:     true,
			errContains: "\n",
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
	validServerNames := []string{"example.com"}
	validPrivateKey := "valid-key"
	validShortIds := []string{""}

	tests := []struct {
		name        string
		settings    SrvInbStreamRealitySettings
		wantErr     bool
		errContains string
	}{
		{
			name: "invalid xver",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: validServerNames,
				PrivateKey:  validPrivateKey,
				Xver:        1,
				ShortIds:    validShortIds,
			},
			wantErr:     true,
			errContains: "xver is '1' while only 0 is supported",
		},
		{
			name: "empty privateKey",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: validServerNames,
				PrivateKey:  "",
				ShortIds:    validShortIds,
			},
			wantErr:     true,
			errContains: "privateKey cannot be empty",
		},
		{
			name: "empty shortIds array",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: validServerNames,
				PrivateKey:  validPrivateKey,
				ShortIds:    []string{},
			},
			wantErr:     true,
			errContains: "there are 0 elements while there must have exactly one element",
		},
		{
			name: "too many shortIds in the array",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: validServerNames,
				PrivateKey:  validPrivateKey,
				ShortIds:    []string{"123", "456"},
			},
			wantErr:     true,
			errContains: "there are 2 elements while there must have exactly one element",
		},
		{
			name: "non-empty shortId",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: validServerNames,
				PrivateKey:  "test-key",
				ShortIds:    []string{"123"},
			},
			wantErr:     true,
			errContains: "shortId is '123' while it must be empty",
		},

		// IsDestValid propagation
		{
			name: "invalid dest",
			settings: SrvInbStreamRealitySettings{
				Dest:        "invalid",
				ServerNames: []string{"invalid"},
				PrivateKey:  validPrivateKey,
				ShortIds:    validShortIds,
			},
			wantErr:     true,
			errContains: "dest is 'invalid' which is not a valid reality dest",
		},

		// ValidateServerNames propagation
		{
			name: "invalid server names",
			settings: SrvInbStreamRealitySettings{
				Dest:        validDomain,
				ServerNames: []string{"wrong.com"},
				PrivateKey:  validPrivateKey,
				ShortIds:    validShortIds,
			},
			wantErr:     true,
			errContains: "serverName 'wrong.com' does not match the domain",
		},

		// Multiple errors
		{
			name: "multiple errors",
			settings: SrvInbStreamRealitySettings{
				Dest:        "invalid",
				ServerNames: []string{"wrong.com"},
				PrivateKey:  "",
				ShortIds:    []string{"123"},
			},
			wantErr:     true,
			errContains: "\n",
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
		{"domain with wildcard", SrvInbStreamRealitySettings{
			Dest: "*.example.com:443", ServerNames: []string{"*.example.com"},
		}, true, "wildcards are not suppported"},
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
		ShortIds:    []string{""},
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
			errContains: "only 'reality' is supported",
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

		// Multiple errors
		{
			name: "multiple errors with propagation",
			settings: SrvInbStreamSettings{
				Network:         "wrong",
				Security:        "reality",
				RealitySettings: invalidReality,
			},
			wantErr:     true,
			errContains: "only 'raw' or 'tcp' (which are interchangeable) are supported\nserverName '' does not match the domain",
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

func TestSrvInbound_Validate(t *testing.T) {
	// Helper valid configurations
	validSniffing := SrvInbSniffing{
		Enabled:      true,
		DestOverride: []string{"http"},
	}
	validSettings := SrvInbSettings{
		Decryption: "none",
	}
	validStreamSettings := &SrvInbStreamSettings{
		RealitySettings: SrvInbStreamRealitySettings{
			Dest:        "example.com:443",
			ServerNames: []string{"example.com"},
			PrivateKey:  "valid-key",
			ShortIds:    []string{""},
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
			errContains: "inbound.sniffing.destOverride shall be set",
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
					Decryption: "none",
					Password:   "short",
					Network:    "tcp",
				},
			},
			wantErr:     true,
			errContains: "cannot set inbound.settings.decryption together",
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

		// Multiple errors
		{
			name: "multiple errors",
			inbound: SrvInbound{
				Protocol: "vless",
				Tag:      "vless-in",
				Port:     443,
				Listen:   externalIPv4,
				Sniffing: SrvInbSniffing{
					Enabled:      true,
					DestOverride: []string{},
				},
				Settings: SrvInbSettings{
					Decryption: "none",
					Password:   "short",
					Network:    "tcp",
				},
			},
			wantErr:     true,
			errContains: "\n",
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
		{
			name: "multiple errors",
			peer: SrvOutboundSettingsPeer{
				Endpoint:  "",
				PublicKey: "",
			},
			wantErr:     true,
			errContains: "\n",
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
		{
			name: "multiple errors",
			settings: SrvOutbSettings{
				SecretKey:      "",
				Address:        []string{},
				Peers:          []SrvOutboundSettingsPeer{validPeer},
				Mtu:            1199,
				Reserved:       []int{1},
				Workers:        0,
				DomainStrategy: "",
			},
			wantErr:     true,
			errContains: "\n",
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

		// Multiple errors
		{
			name: "multiple errors",
			outbound: SrvOutbound{
				Protocol: "",
				Tag:      "",
			},
			wantErr:     true,
			errContains: "\n",
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

		// Multiple errors
		{
			name: "multiple errors",
			rule: SrvRoutingRule{
				Type:        "",
				OutboundTag: "",
			},
			wantErr:     true,
			errContains: "\n",
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

func TestSrvRouting_Validate(t *testing.T) {
	// Helper valid rule
	validRule := SrvRoutingRule{
		Type:        "field",
		OutboundTag: "valid-outbound",
	}

	tests := []struct {
		name        string
		routing     SrvRouting
		wantErr     bool
		errContains string
	}{
		// Valid cases
		{
			name: "valid minimal routing",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{validRule},
				DomainStrategy: "AsIs",
			},
			wantErr: false,
		},
		{
			name: "valid with multiple rules",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{validRule, validRule},
				DomainStrategy: "IPIfNonMatch",
			},
			wantErr: false,
		},
		{
			name: "valid IPOnDemand strategy",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{validRule},
				DomainStrategy: "IPOnDemand",
			},
			wantErr: false,
		},

		// Rules validation
		{
			name: "empty rules array",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{},
				DomainStrategy: "AsIs",
			},
			wantErr:     true,
			errContains: "routing.rules array cannot be empty",
		},
		{
			name: "invalid rule in rules",
			routing: SrvRouting{
				Rules: []SrvRoutingRule{
					{
						Type:        "",
						OutboundTag: "test",
					},
				},
				DomainStrategy: "AsIs",
			},
			wantErr:     true,
			errContains: "routing.rules.type cannot be empty",
		},

		// DomainStrategy validation
		{
			name: "empty domainStrategy",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{validRule},
				DomainStrategy: "",
			},
			wantErr:     true,
			errContains: "routing.domainStrategy cannot be empty",
		},
		{
			name: "invalid domainStrategy",
			routing: SrvRouting{
				Rules:          []SrvRoutingRule{validRule},
				DomainStrategy: "InvalidStrategy",
			},
			wantErr:     true,
			errContains: "invalid routing.domainStrategy 'InvalidStrategy'",
		},

		// Multiple errors
		{
			name: "multiple errors",
			routing: SrvRouting{
				Rules: []SrvRoutingRule{
					{
						Type:        "",
						OutboundTag: "",
					},
				},
				DomainStrategy: "",
			},
			wantErr:     true,
			errContains: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.routing.Validate()
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
