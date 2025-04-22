package main

type ClientInbound struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ClientOutboundSettingsServer struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
}

type ClientOutboundSettings struct {
	Servers []ClientOutboundSettingsServer `json:"servers"`
}

type ClientOutbound struct {
	Protocol string                 `json:"protocol"`
	Settings ClientOutboundSettings `json:"settings"`
	Tag      string                 `json:"tag"`
}

type ClientRoutingRule struct {
	Type        string `json:"type"`
	OutboundTag string `json:"outboundTag"`
	Network     string `json:"network"`
}

type ClientRouting struct {
	Rules          []ClientRoutingRule `json:"rules"`
	DomainStrategy string              `json:"domainStrategy"`
}

type ClientConfig struct {
	Log       Log              `json:"log"`
	Inbounds  []ClientInbound  `json:"inbounds"`
	Outbounds []ClientOutbound `json:"outbounds"`
	Routing   ClientRouting    `json:"routing"`
}
