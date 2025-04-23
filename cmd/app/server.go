package main

import (
	"errors"
	"fmt"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type Log struct {
	Loglevel string `json:"loglevel"`
}

func (l *Log) Validate() error {
	switch l.Loglevel {
	case "debug", "info", "warning", "error", "none": // Acceptable
	case "":
		return fmt.Errorf(
			"xray server config must have the logger block with loglevel set")
	default:
		return fmt.Errorf(`xray server config must have the logger block with ` +
			`loglevel set; allowed values: "debug", "info", "warning", "error", "none"`)
	}
	return nil
}

type SrvInbSniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

func (s *SrvInbSniffing) Validate() error {
	if s.Enabled && len(s.DestOverride) == 0 {
		return errors.New(
			"xray server config must have the inbound block with sniffing " +
				"enabled and destOverride set")
	}

	for _, dest := range s.DestOverride {
		switch dest {
		case "http", "tls", "quic":
		default:
			return fmt.Errorf(`xray server config must have the inbound block ` +
				`with sniffing enabled and destOverride set; allowed values: ` +
				`"http", "tls", "quic"`)
		}
	}

	return nil
}

type SrvInbSettingsClient struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Flow  string `json:"flow,omitempty"`
}

func (s *SrvInbSettingsClient) Validate() error {
	if !utils.IsValidUUID(s.ID) {
		return fmt.Errorf("client id is '%s' which is not a valid UUID", s.ID)
	}

	if s.Email == "" {
		return errors.New("client email shall not be empty")
	}

	switch s.Flow {
	case "xtls-rprx-vision":
	case "":
	default:
		return fmt.Errorf("client flow is '%s' while only xtls-rprx-vision is "+
			"allowed (or none at all)", s.Flow)
	}
	return nil
}

type SrvInbSettings struct {
	Clients    *[]SrvInbSettingsClient `json:"clients,omitempty"`
	Decryption string                  `json:"decryption,omitempty"`
	Method     string                  `json:"method,omitempty"`
	Password   string                  `json:"password,omitempty"`
	Network    string                  `json:"network,omitempty"`
}

func (s *SrvInbSettings) Validate() error {
	if s.Clients == nil || len(*s.Clients) == 0 {
		return errors.New("client list should not be empty")
	}

	for _, client := range *s.Clients {
		if err := client.Validate(); err != nil {
			return err
		}
	}

	switch s.Decryption {
	case "none":
	case "":
		return errors.New("decryption cannot be left empty")
	default:
		return fmt.Errorf(
			"decryption is '%s' while only 'none' is allowed", s.Decryption)
	}

	switch s.Method {
	case "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305":
	case "":
	default:
		return fmt.Errorf("method is '%s' while only the following options are "+
			"allowed: '2022-blake3-aes-128-gcm' | '2022-blake3-aes-256-gcm' | "+
			"'2022-blake3-chacha20-poly1305'", s.Method)
	}

	if len(s.Password) < 16 {
		return errors.New("password is too short")
	}

	switch s.Network {
	case "tcp", "udp", "tcp,udp":
	case "":
		return errors.New("network cannot be left empty")
	default:
		return fmt.Errorf("network is '%s' while only the following options are "+
			"allowed: 'tcp' | 'udp' | 'tcp,udp'", s.Network)
	}

	return nil
}

type SrvInbStreamRealitySettings struct {
	Show         bool     `json:"show"`
	Dest         string   `json:"dest"`
	Xver         int      `json:"xver"`
	ServerNames  []string `json:"serverNames"`
	PrivateKey   string   `json:"privateKey"`
	MinClientVer string   `json:"minClientVer"`
	MaxClientVer string   `json:"maxClientVer"`
	MaxTimeDiff  int      `json:"maxTimeDiff"`
	ShortIds     []string `json:"shortIds"`
}

type SrvInbKCPHeader struct {
	Type string `json:"type"`
}

type SrvInbStreamKCPSettings struct {
	Mtu              int             `json:"mtu"`
	Tti              int             `json:"tti"`
	UplinkCapacity   int             `json:"uplinkCapacity"`
	DownlinkCapacity int             `json:"downlinkCapacity"`
	Congestion       bool            `json:"congestion"`
	ReadBufferSize   int             `json:"readBufferSize"`
	WriteBufferSize  int             `json:"writeBufferSize"`
	Header           SrvInbKCPHeader `json:"header"`
	Seed             string          `json:"seed"`
}

type SrvInbStreamSettings struct {
	Network         string                       `json:"network"`
	Security        string                       `json:"security,omitempty"`
	RealitySettings *SrvInbStreamRealitySettings `json:"realitySettings,omitempty"`
	KcpSettings     *SrvInbStreamKCPSettings     `json:"kcpSettings,omitempty"`
}

type SrvInbound struct {
	Protocol       string                `json:"protocol"`
	Tag            string                `json:"tag"`
	Port           int                   `json:"port"`
	Listen         string                `json:"listen,omitempty"`
	Sniffing       SrvInbSniffing        `json:"sniffing"`
	Settings       SrvInbSettings        `json:"settings"`
	StreamSettings *SrvInbStreamSettings `json:"streamSettings,omitempty"`
}

func (i *SrvInbound) Validate() error {
	switch i.Protocol {
	case "vless", "shadowsocks": // Acceptable
	case "":
		return fmt.Errorf("inbound.protocol cannot be empty")
	default:
		return fmt.Errorf(
			"only vless and shadowsocks protocols are supported")
	}
	return nil
}

type SrvOutboundSettingsPeer struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"publicKey"`
}

type SrvOutbSettings struct {
	SecretKey      string                    `json:"secretKey"`
	Address        []string                  `json:"address"`
	Peers          []SrvOutboundSettingsPeer `json:"peers"`
	Mtu            int                       `json:"mtu"`
	Reserved       []int                     `json:"reserved"`
	Workers        int                       `json:"workers"`
	DomainStrategy string                    `json:"domainStrategy"`
}

type SrvOutbound struct {
	Protocol string           `json:"protocol"`
	Tag      string           `json:"tag"`
	Settings *SrvOutbSettings `json:"settings,omitempty"`
}

type SrvRoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	Protocol    string   `json:"protocol,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
}

type SrvRouting struct {
	Rules          []SrvRoutingRule `json:"rules"`
	DomainStrategy string           `json:"domainStrategy"`
}

type ServerConfig struct {
	Log       Log           `json:"log"`
	Inbounds  []SrvInbound  `json:"inbounds"`
	Outbounds []SrvOutbound `json:"outbounds"`
	Routing   SrvRouting    `json:"routing"`
}

func (c *ServerConfig) Validate() error {
	var errs utils.Errors

	err := c.Log.Validate()
	if err != nil {
		errs.Append(err)
	}

	if len(c.Inbounds) == 0 {
		errs.Append(errors.New("xray server config must have at least one inbound"))
	}

	shadowsocksExists := false
	for _, inbound := range c.Inbounds {
		if inbound.Protocol == "shadowsocks" {
			shadowsocksExists = true
			break
		}
	}
	if !shadowsocksExists {
		errs.Append(errors.New("xray server config must have at least one inbound " +
			"with shadowsocks protocol since it will be required for the warp " +
			"verification client"))
	}

	for _, inbound := range c.Inbounds {
		err = inbound.Validate()
		if err != nil {
			errs.Append(err)
		}
	}
	return nil
}
