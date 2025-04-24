package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

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

func (s *SrvInbStreamRealitySettings) IsDestValid() bool {
	// A special case: Reality dest can be set to '1.1.1.1:443'
	if s.Dest == "1.1.1.1:443" {
		return true
	}

	// Split the string into domain and port parts
	parts := strings.Split(s.Dest, ":")
	if len(parts) != 2 {
		return false
	}

	// Check the port is exactly "443"
	if parts[1] != "443" {
		return false
	}

	// Validate the domain part
	domain := parts[0]
	if domain == "" {
		return false
	}

	// Regular expression for domain validation
	// This allows:
	// - letters a-z (case insensitive)
	// - digits 0-9
	// - hyphens (but not at start or end)
	// - dots (but not at start or end)
	// - at least one dot (for subdomains)
	domainRegex := `^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(domainRegex, domain)
	if err != nil {
		return false
	}

	return matched
}

func (s *SrvInbStreamRealitySettings) ValidateServerNames() error {
	if len(s.ServerNames) != 1 {
		return errors.New("serverNames must have exactly one element")
	}

	if strings.Contains(s.ServerNames[0], "*") {
		return errors.New("wildcards are not suppported in serverNames")
	}

	domain := strings.Split(s.Dest, ":")[0]
	if domain == "1.1.1.1" {
		if s.ServerNames[0] != "" {
			return errors.New("when the dest is '1.1.1.1:443', serverName must be empty")
		}
	} else {
		if s.ServerNames[0] != domain {
			return fmt.Errorf("serverName '%s' does not match the domain in the dest "+
				"'%s'", s.ServerNames[0], s.Dest)
		}
	}
	return nil
}

func (s *SrvInbStreamRealitySettings) Validate() error {
	if !s.IsDestValid() {
		return fmt.Errorf("dest is '%s' which is not a valid reality dest: "+
			"it should be either '1.1.1.1:443' or a valid domain with port 443, e.g.: "+
			"'example.com:443'", s.Dest)
	}

	if s.Xver != 0 {
		return fmt.Errorf("xver is '%d' while only 0 is supported", s.Xver)
	}

	err := s.ValidateServerNames()
	if err != nil {
		return err
	}

	if s.PrivateKey == "" {
		return errors.New("privateKey cannot be empty")
	}

	if len(s.ShortIds) != 0 && s.ShortIds[0] != "" {
		return errors.New("shortIds must be empty")
	}

	return nil
}

type SrvInbStreamSettings struct {
	Network         string                      `json:"network"`
	Security        string                      `json:"security,omitempty"`
	RealitySettings SrvInbStreamRealitySettings `json:"realitySettings"`
}

func (s *SrvInbStreamSettings) Validate() error {
	switch s.Network {
	case "raw", "tcp":
	case "":
		return errors.New("network cannot be empty")
	default:
		return fmt.Errorf("network is '%s' while only 'raw' or 'tcp' (which are "+
			"interchangeable) are supported", s.Network)
	}

	switch s.Security {
	case "reality":
		return s.RealitySettings.Validate()
	case "":
		return errors.New("security cannot be empty")
	default:
		return errors.New("only 'reality' security is supported")
	}
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
			"inbound.protocol: only vless and shadowsocks protocols are supported")
	}

	if i.Tag == "" {
		return errors.New("inbound.tag cannot be empty")
	}

	if i.Protocol == "vless" && i.Port != 443 {
		return errors.New("inbound.port: vless protocol only supports port 443")
	}

	if i.Protocol == "vless" && !utils.IsExternalIPv4(i.Listen) {
		return errors.New("inbound.listen shall be an external IPv4 address for the " +
			"vless protocol")
	}

	err := i.Sniffing.Validate()
	if err != nil {
		return err
	}

	err = i.Settings.Validate()
	if err != nil {
		return err
	}

	if i.StreamSettings != nil {
		err = i.StreamSettings.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type SrvOutboundSettingsPeer struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"publicKey"`
}

func (p *SrvOutboundSettingsPeer) Validate() error {
	if p.PublicKey == "" {
		return errors.New("publicKey cannot be empty")
	}

	if !utils.IsValidEndpoint(p.Endpoint) {
		return fmt.Errorf("endpoint '%s' is not a valid endpoint", p.Endpoint)
	}

	return nil
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

func (s *SrvOutbSettings) Validate() error {
	if s.SecretKey == "" {
		return errors.New("outbound.settings.secretKey cannot be empty")
	}

	if len(s.Address) == 0 {
		return errors.New("outbound.settings.address array cannot be empty")
	}

	for _, addr := range s.Address {
		if !utils.IsValidIPOrCIDR(addr) {
			return fmt.Errorf(
				"outbound.settings.address '%s' is not a valid address", addr)
		}
	}

	for _, peer := range s.Peers {
		err := peer.Validate()
		if err != nil {
			return err
		}
	}

	if s.Mtu < 1280 || s.Mtu > 1500 {
		return fmt.Errorf("outbound.settings.mtu must be between 1280 and 1500")
	}

	if len(s.Reserved) == 0 {
		return errors.New("outbound.settings.reserved array cannot be empty")
	}

	if s.Workers < 1 {
		return errors.New("outbound.settings.workers must be at least 1")
	}

	switch s.DomainStrategy {
	case "ForceIPv6v4", "ForceIPv6", "ForceIPv4v6", "ForceIPv4", "ForceIP":
	case "":
		return errors.New("outbound.settings.domainStrategy cannot be empty")
	default:
		return fmt.Errorf("outbound.settings.domainStrategy is '%s' while only "+
			"ForceIPv6v4, ForceIPv6, ForceIPv4v6, ForceIPv4, and ForceIP are supported",
			s.DomainStrategy)
	}

	return nil
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

	// TODO: Check tag uniqueness
	return nil
}
