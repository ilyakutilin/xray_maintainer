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
	case "debug", "info", "warning", "error", "none":
	case "":
		return errors.New(
			"xray server config must have the logger block with loglevel set")
	default:
		return errors.New(`xray server config must have the logger block with ` +
			`loglevel set; allowed values: "debug", "info", "warning", "error", "none"`)
	}
	return nil
}

type SrvInbSniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

func (s *SrvInbSniffing) Validate() error {
	if !s.Enabled {
		return errors.New("inbound.sniffing must be enabled")
	}

	if len(s.DestOverride) == 0 {
		return errors.New("inbound.sniffing.destOverride shall be set")
	}

	for _, dest := range s.DestOverride {
		switch dest {
		case "http", "tls", "quic":
		default:
			return fmt.Errorf("wrong value for inbound.sniffing.destOverride: '%v'. "+
				"allowed values: 'http', 'tls', 'quic'", dest)
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
	var errs utils.Errors

	if !utils.IsValidUUID(s.ID) {
		errs.Append(fmt.Errorf("inbound.settings.client.id is '%s' which is not "+
			"a valid UUID", s.ID))
	}

	if s.Email == "" {
		errs.Append(errors.New("client email shall not be empty"))
	}

	switch s.Flow {
	case "xtls-rprx-vision":
	case "":
	default:
		errs.Append(fmt.Errorf("inbound.settings.client.flow is '%s' while only "+
			"xtls-rprx-vision is allowed (or none at all)", s.Flow))
	}

	if len(errs) > 0 {
		return errs
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
	var errs utils.Errors

	if s.Clients != nil && len(*s.Clients) != 0 {
		for _, client := range *s.Clients {
			if err := client.Validate(); err != nil {
				errs.Append(err)
			}
		}
	}

	// The logic is that Decryption only applies to vless inbound, while Method,
	// Password, and Network only apply to shadowsocks inbound. Therefore if Decryption
	// is present, the rest shall be empty. If any of the rest is present, Decryption
	// shall be empty. Also, Method, Password, and Network shall all be set if
	// at least one of them is set.
	allSSFieldsEmpty := s.Method == "" && s.Password == "" && s.Network == ""
	anySSFieldsEmpty := s.Method == "" || s.Password == "" || s.Network == ""

	if s.Decryption != "" && !allSSFieldsEmpty {
		return errors.New("cannot set inbound.settings.decryption together with " +
			"inbound.settings.method / password / network")
	}

	if s.Decryption == "" && allSSFieldsEmpty {
		return errors.New("either inbound.settings.decryption or " +
			"the inbound.settings.method, password, and network shall be set")
	}

	if !allSSFieldsEmpty {
		if anySSFieldsEmpty {
			return errors.New("inbound.settings.method, password, and network " +
				"shall all be set if at least one of them is set")
		}

		switch s.Method {
		case "2022-blake3-aes-128-gcm":
		case "2022-blake3-aes-256-gcm":
		case "2022-blake3-chacha20-poly1305":
		default:
			errs.Append(fmt.Errorf("inbound.settings.method is '%s' while only "+
				"the following options are allowed: '2022-blake3-aes-128-gcm' | "+
				"'2022-blake3-aes-256-gcm' | '2022-blake3-chacha20-poly1305', ",
				s.Method))
		}

		if len(s.Password) < 16 {
			errs.Append(errors.New("inbound.settings.password is too short"))
		}

		switch s.Network {
		case "tcp", "udp", "tcp,udp":
		default:
			errs.Append(fmt.Errorf("inbound.settings.network is '%s' while only "+
				"the following options are allowed: 'tcp' | 'udp' | 'tcp,udp', ",
				s.Network))
		}

	} else {
		switch s.Decryption {
		case "none":
		default:
			errs.Append(fmt.Errorf("inbound.settings.decryption is '%s' while only "+
				"'none' is allowed", s.Decryption))
		}
	}

	if len(errs) > 0 {
		return errs
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
	var errs utils.Errors

	if !s.IsDestValid() {
		errs.Append(fmt.Errorf("inbound.streamSettings.realitySettings.dest is '%s' "+
			"which is not a valid reality dest: it should be either '1.1.1.1:443' "+
			"or a valid domain with port 443, e.g.: 'example.com:443'", s.Dest))
	}

	if s.Xver != 0 {
		errs.Append(fmt.Errorf("inbound.streamSettings.realitySettings.xver is '%d' "+
			"while only 0 is supported", s.Xver))
	}

	err := s.ValidateServerNames()
	if err != nil {
		errs.Append(err)
	}

	if s.PrivateKey == "" {
		errs.Append(errors.New("inbound.streamSettings.realitySettings.privateKey " +
			"cannot be empty"))
	}

	if len(s.ShortIds) != 1 {
		errs.Append(fmt.Errorf("inbound.streamSettings.realitySettings.shortIds: "+
			"there are %d elements while there must have exactly one element "+
			"and it shall be an empty string", len(s.ShortIds)))
		return errs
	}

	if s.ShortIds[0] != "" {
		errs.Append(fmt.Errorf("inbound.streamSettings.realitySettings.shortId "+
			"is '%v' while it must be empty", s.ShortIds[0]))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

type SrvInbStreamSettings struct {
	Network         string                      `json:"network"`
	Security        string                      `json:"security"`
	RealitySettings SrvInbStreamRealitySettings `json:"realitySettings"`
}

func (s *SrvInbStreamSettings) Validate() error {
	var errs utils.Errors

	switch s.Network {
	case "raw", "tcp":
	case "":
		errs.Append(errors.New("inbound.streamSettings.network cannot be empty"))
	default:
		errs.Append(fmt.Errorf("inbound.streamSettings.network is '%s' while only "+
			"'raw' or 'tcp' (which are interchangeable) are supported", s.Network))
	}

	switch s.Security {
	case "reality":
		err := s.RealitySettings.Validate()
		if err != nil {
			errs.Append(err)
		}
	case "":
		errs.Append(errors.New("inbound.streamSettings.security cannot be empty"))
	default:
		errs.Append(errors.New("inbound.streamSettings.security: only 'reality' " +
			"is supported"))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
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
	var errs utils.Errors

	switch i.Protocol {
	case "vless", "shadowsocks":
	case "":
		errs.Append(fmt.Errorf("inbound.protocol cannot be empty"))
	default:
		errs.Append(fmt.Errorf(
			"inbound.protocol: only vless and shadowsocks protocols are supported"))
	}

	if i.Tag == "" {
		errs.Append(errors.New("inbound.tag cannot be empty"))
	}

	if i.Protocol == "vless" && i.Port != 443 {
		errs.Append(errors.New("inbound.port: vless protocol only supports port 443"))
	}

	if i.Protocol == "vless" && !utils.IsExternalIPv4(i.Listen) {
		errs.Append(errors.New("inbound.listen shall be an external IPv4 address for the " +
			"vless protocol"))
	}

	err := i.Sniffing.Validate()
	if err != nil {
		errs.Append(err)
	}

	err = i.Settings.Validate()
	if err != nil {
		errs.Append(err)
	}

	if i.StreamSettings != nil {
		err = i.StreamSettings.Validate()
		if err != nil {
			errs.Append(err)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

type SrvOutboundSettingsPeer struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"publicKey"`
}

func (p *SrvOutboundSettingsPeer) Validate() error {
	var errs utils.Errors

	if p.PublicKey == "" {
		errs.Append(errors.New("outbound.settings.peer.publicKey cannot be empty"))
	}

	if !utils.IsValidEndpoint(p.Endpoint) {
		errs.Append(fmt.Errorf("outbound.settings.peer.endpoint '%s' "+
			"is not a valid endpoint", p.Endpoint))
	}

	if len(errs) > 0 {
		return errs
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
	var errs utils.Errors

	if s.SecretKey == "" {
		errs.Append(errors.New("outbound.settings.secretKey cannot be empty"))
	}

	if len(s.Address) == 0 {
		errs.Append(errors.New("outbound.settings.address array cannot be empty"))
	}

	for _, addr := range s.Address {
		if !utils.IsValidIPOrCIDR(addr) {
			errs.Append(fmt.Errorf(
				"outbound.settings.address '%s' is not a valid address", addr))
		}
	}

	for _, peer := range s.Peers {
		err := peer.Validate()
		if err != nil {
			errs.Append(err)
		}
	}

	if s.Mtu < 1280 || s.Mtu > 1500 {
		errs.Append(fmt.Errorf("outbound.settings.mtu must be between 1280 and 1500"))
	}

	if len(s.Reserved) == 0 {
		errs.Append(errors.New("outbound.settings.reserved array cannot be empty"))
	}

	if s.Workers < 1 {
		errs.Append(errors.New("outbound.settings.workers must be at least 1"))
	}

	switch s.DomainStrategy {
	case "ForceIPv6v4", "ForceIPv6", "ForceIPv4v6", "ForceIPv4", "ForceIP":
	case "":
		errs.Append(errors.New("outbound.settings.domainStrategy cannot be empty"))
	default:
		errs.Append(fmt.Errorf("outbound.settings.domainStrategy is '%s' while only "+
			"ForceIPv6v4, ForceIPv6, ForceIPv4v6, ForceIPv4, and ForceIP are supported",
			s.DomainStrategy))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

type SrvOutbound struct {
	Protocol string           `json:"protocol"`
	Tag      string           `json:"tag"`
	Settings *SrvOutbSettings `json:"settings,omitempty"`
}

func (o *SrvOutbound) Validate() error {
	var errs utils.Errors

	switch o.Protocol {
	case "blackhole", "freedom", "vless", "shadowsocks", "wireguard":
	case "":
		errs.Append(errors.New("outbound.protocol cannot be empty"))
	default:
		errs.Append(fmt.Errorf("invalid outbound.protocol '%v': only 'blackhole', "+
			"'freedom', 'vless', 'shadowsocks', and 'wireguard' protocold are "+
			"supported", o.Protocol))
	}

	if o.Tag == "" {
		errs.Append(errors.New("outbound.tag cannot be empty"))
	}

	if o.Protocol == "wireguard" && o.Settings == nil {
		errs.Append(errors.New("outbound.settings cannot be empty " +
			"for wireguard protocol"))
	}

	if o.Settings != nil {
		err := o.Settings.Validate()
		if err != nil {
			errs.Append(err)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

type SrvRoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	Protocol    string   `json:"protocol,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
}

func (r *SrvRoutingRule) Validate() error {
	var errs utils.Errors

	switch r.Type {
	case "field":
	case "":
		errs.Append(errors.New("routing.rules.type cannot be empty"))
	default:
		errs.Append(fmt.Errorf("invalid routing.rules.type '%v': only 'field', "+
			"is supported", r.Type))
	}

	if r.OutboundTag == "" {
		errs.Append(errors.New("routing.rules.outboundTag cannot be empty"))
	}

	// There are specific rules that apply to the Protocol, IP and Domain fields
	// that are set by the xray core devs. It would be way too verbose to validate
	// each of them so please refer to their website for guidelines:
	// https://xtls.github.io/en/config/routing.html#ruleobject

	if len(errs) > 0 {
		return errs
	}

	return nil
}

type SrvRouting struct {
	Rules          []SrvRoutingRule `json:"rules"`
	DomainStrategy string           `json:"domainStrategy"`
}

func (r *SrvRouting) Validate() error {
	var errs utils.Errors

	if len(r.Rules) == 0 {
		errs.Append(errors.New("routing.rules array cannot be empty"))
		return errs
	}

	for _, rule := range r.Rules {
		err := rule.Validate()
		if err != nil {
			errs.Append(err)
		}
	}

	switch r.DomainStrategy {
	case "AsIs", "IPIfNonMatch", "IPOnDemand":
	case "":
		errs.Append(errors.New("routing.domainStrategy cannot be empty"))
	default:
		errs.Append(fmt.Errorf("invalid routing.domainStrategy '%v': only 'AsIs', "+
			"'IPIfNonMatch', and 'IPOnDemand' are supported", r.DomainStrategy))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
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
