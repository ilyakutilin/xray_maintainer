package main

type ServerConfig struct {
	Log struct {
		Loglevel string `json:"loglevel"`
	} `json:"log"`
	Inbounds []struct {
		Protocol string `json:"protocol"`
		Tag      string `json:"tag"`
		Port     int    `json:"port"`
		Listen   string `json:"listen,omitempty"`
		Sniffing struct {
			Enabled      bool     `json:"enabled"`
			DestOverride []string `json:"destOverride"`
		} `json:"sniffing"`
		Settings struct {
			Clients *[]struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Flow  string `json:"flow,omitempty"`
			} `json:"clients,omitempty"`
			Decryption string `json:"decryption,omitempty"`
			Method     string `json:"method,omitempty"`
			Password   string `json:"password,omitempty"`
			Network    string `json:"network,omitempty"`
		} `json:"settings"`
		StreamSettings *struct {
			Network         string `json:"network"`
			Security        string `json:"security,omitempty"`
			RealitySettings *struct {
				Show         bool     `json:"show"`
				Dest         string   `json:"dest"`
				Xver         int      `json:"xver"`
				ServerNames  []string `json:"serverNames"`
				PrivateKey   string   `json:"privateKey"`
				MinClientVer string   `json:"minClientVer"`
				MaxClientVer string   `json:"maxClientVer"`
				MaxTimeDiff  int      `json:"maxTimeDiff"`
				ShortIds     []string `json:"shortIds"`
			} `json:"realitySettings,omitempty"`
			KcpSettings *struct {
				Mtu              int  `json:"mtu"`
				Tti              int  `json:"tti"`
				UplinkCapacity   int  `json:"uplinkCapacity"`
				DownlinkCapacity int  `json:"downlinkCapacity"`
				Congestion       bool `json:"congestion"`
				ReadBufferSize   int  `json:"readBufferSize"`
				WriteBufferSize  int  `json:"writeBufferSize"`
				Header           struct {
					Type string `json:"type"`
				} `json:"header"`
				Seed string `json:"seed"`
			} `json:"kcpSettings,omitempty"`
		} `json:"streamSettings,omitempty"`
	} `json:"inbounds"`
	Outbounds []struct {
		Protocol string `json:"protocol"`
		Tag      string `json:"tag"`
		Settings *struct {
			SecretKey string   `json:"secretKey"`
			Address   []string `json:"address"`
			Peers     []struct {
				Endpoint  string `json:"endpoint"`
				PublicKey string `json:"publicKey"`
			} `json:"peers"`
			Mtu            int    `json:"mtu"`
			Reserved       []int  `json:"reserved"`
			Workers        int    `json:"workers"`
			DomainStrategy string `json:"domainStrategy"`
		} `json:"settings,omitempty"`
	} `json:"outbounds"`
	Routing struct {
		Rules []struct {
			Type        string   `json:"type"`
			OutboundTag string   `json:"outboundTag"`
			Protocol    string   `json:"protocol,omitempty"`
			Domain      []string `json:"domain,omitempty"`
			IP          []string `json:"ip,omitempty"`
		} `json:"rules"`
		DomainStrategy string `json:"domainStrategy"`
	} `json:"routing"`
}

func (c *ServerConfig) Validate() error {
	// TODO: Implement
	return nil
}
