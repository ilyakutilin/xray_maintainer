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
