package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ilyakutilin/xray_maintainer/messages"
	"github.com/ilyakutilin/xray_maintainer/utils"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type XrayServer struct {
	IP             string `koanf:"ip"`
	ConfigFilePath string
}

type XrayClient struct {
	// Only shadowsocks is supported so the value is not taken from yaml, it is always
	// taken from the defaults
	ServerProtocol string
	Port           int `koanf:"port"`
	ConfigFilePath string
}

type Xray struct {
	Server             XrayServer `koanf:"server"`
	Client             XrayClient `koanf:"client"`
	ExecutableFilePath string
}

type Repo struct {
	Name           string `koanf:"name"`
	ReleaseInfoURL string `koanf:"release_info_url"`
	DownloadURL    string `koanf:"download_url"`
	Filename       string `koanf:"filename"`
	Executable     bool   `koanf:"executable"`
}

type Messages struct {
	EmailSender    messages.EmailSender    `koanf:"email"`
	TelegramSender messages.TelegramSender `koanf:"telegram"`
	StreamSender   messages.StreamSender
	MainSender     messages.CompositeSender
}

type Config struct {
	Debug        bool     `koanf:"debug"`
	Workdir      string   `koanf:"workdir"`
	Xray         Xray     `koanf:"xray"`
	Repos        []Repo   `koanf:"repos"`
	Messages     Messages `koanf:"messages"`
	IPCheckerURL string   `koanf:"ip_checker_url"`
}

var defaults = Config{
	Debug:   true,
	Workdir: "/opt/xray/",
	Xray: Xray{
		Server: XrayServer{
			// No default for Server IP as it shall be explicitly set by the user
			IP: "",
		},
		Client: XrayClient{
			ServerProtocol: "shadowsocks",
			Port:           10801,
		},
	},
	Repos: []Repo{
		{
			Name:           "geoip",
			ReleaseInfoURL: "https://api.github.com/repos/v2fly/geoip/releases/latest",
			DownloadURL:    "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",
			Filename:       "geoip.dat",
			Executable:     false,
		},
		{
			Name:           "geosite",
			ReleaseInfoURL: "https://api.github.com/repos/v2fly/geoip/releases/latest",
			DownloadURL:    "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat",
			Filename:       "geosite.dat",
			Executable:     false,
		},
		{
			Name:           "xray-core",
			ReleaseInfoURL: "https://api.github.com/repos/XTLS/Xray-core/releases/latest",
			DownloadURL:    "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip",
			Filename:       "xray",
			Executable:     true,
		},
		{
			Name:           "cf_cred_generator",
			ReleaseInfoURL: "https://api.github.com/repos/badafans/warp-reg/releases/latest",
			DownloadURL:    "https://github.com/badafans/warp-reg/releases/latest/download/main-linux-amd64",
			Filename:       "cf_cred_generator",
			Executable:     true,
		},
	},
	Messages: Messages{
		// EmailSender and TelegramSender settings shall be provided by the user in full
		// StreamSender has no settings
		StreamSender: messages.StreamSender{},
	},
	IPCheckerURL: "http://ip-api.com/json/",
}

// Loads configuration
func loadConfig() (*Config, error) {
	var k = koanf.New(".")

	if err := k.Load(structs.Provider(defaults, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("error loading the default values for the config: %w", err)
	}

	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config values from yaml: %w", err)
	}

	cfg := &Config{}

	k.Unmarshal("", cfg)

	var err error

	if cfg.Xray.Server.IP == "" {
		return nil, errors.New("xray server IP should be set")
	}

	cfg.Workdir, err = utils.ExpandPath(cfg.Workdir)
	if err != nil {
		return nil, fmt.Errorf("error expanding the workdir path: %w", err)
	}

	cfg.Xray.Server.ConfigFilePath = filepath.Join(cfg.Workdir, "server-config.json")
	cfg.Xray.Client.ConfigFilePath = filepath.Join(cfg.Workdir, "client-config.json")

	var xrayExecutableFileName string
	for _, repo := range cfg.Repos {
		if repo.Name == "xray_core" {
			xrayExecutableFileName = repo.Filename
		}
	}
	if xrayExecutableFileName == "" {
		return nil, errors.New("the name for the xray core executable has not " +
			"been set in the config. Please set the filename for the xray " +
			"executable")
	}
	cfg.Xray.ExecutableFilePath = filepath.Join(cfg.Workdir, xrayExecutableFileName)

	rawSenders := []messages.Sender{
		&cfg.Messages.EmailSender,
		&cfg.Messages.TelegramSender,
	}

	var validSenders []messages.Sender

	for _, sender := range rawSenders {
		if err := sender.Validate(); err != nil {
			return nil, fmt.Errorf("the sender failed validation and will not be "+
				"included in the senders list: %w", err)
		}
		validSenders = append(validSenders, sender)
	}
	cfg.Messages.MainSender = messages.CompositeSender{
		Senders: validSenders,
	}

	return cfg, nil
}
