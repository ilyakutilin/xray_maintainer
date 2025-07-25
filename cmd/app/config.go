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
	ServiceName    string `koanf:"service_name"`
	ConfigFileName string `koanf:"config_filename"`
	ConfigFilePath string
}

type XrayClient struct {
	// Only shadowsocks is supported so the value is not taken from yaml, it is always
	// taken from the defaults
	ServerProtocol string
	Port           int    `koanf:"port"`
	IPCheckerURL   string `koanf:"ip_checker_url"`
	ConfigFileName string `koanf:"config_filename"`
	ConfigFilePath string
}

type Xray struct {
	Server             XrayServer `koanf:"server"`
	Client             XrayClient `koanf:"client"`
	ExecutableFilePath string
	CFCredFilePath     string
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
	Debug    bool     `koanf:"debug"`
	Workdir  string   `koanf:"workdir"`
	Xray     Xray     `koanf:"xray"`
	Repos    []Repo   `koanf:"repos"`
	Messages Messages `koanf:"messages"`
}

var defaults = Config{
	Debug:   false,
	Workdir: ".",
	Xray: Xray{
		Server: XrayServer{
			// No default for Server IP as it shall be explicitly set by the user
			IP:             "",
			ServiceName:    "xray.service",
			ConfigFileName: "config.json",
		},
		Client: XrayClient{
			ServerProtocol: "shadowsocks",
			Port:           10801,
			IPCheckerURL:   "http://ip-api.com/json/?fields=status,message,isp,org,query",
			ConfigFileName: "client-config.json",
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
			ReleaseInfoURL: "https://api.github.com/repos/v2fly/domain-list-community/releases/latest",
			DownloadURL:    "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat",
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
}

func findFilenameInRepo(repos []Repo, repoName string) (string, error) {
	var fileName string

	for _, repo := range repos {
		if repo.Name == repoName {
			fileName = repo.Filename
		}
	}

	if fileName == "" {
		return "", fmt.Errorf("the name for the %s has not been set in the config. "+
			"Please set the filename for the %s", repoName, repoName)
	}

	return fileName, nil
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

	cfg.Xray.Server.ConfigFilePath = filepath.Join(cfg.Workdir, cfg.Xray.Server.ConfigFileName)
	cfg.Xray.Client.ConfigFilePath = filepath.Join(cfg.Workdir, cfg.Xray.Client.ConfigFileName)

	xrayExecutableFileName, err := findFilenameInRepo(cfg.Repos, "xray-core")
	if err != nil {
		return nil, err
	}
	cfg.Xray.ExecutableFilePath = filepath.Join(cfg.Workdir, xrayExecutableFileName)

	cfCredFileName, err := findFilenameInRepo(cfg.Repos, "cf_cred_generator")
	if err != nil {
		return nil, err
	}
	cfg.Xray.CFCredFilePath = filepath.Join(cfg.Workdir, cfCredFileName)

	rawSenders := []messages.Sender{
		&cfg.Messages.EmailSender,
		&cfg.Messages.TelegramSender,
	}

	var validSenders []messages.Sender

	for _, sender := range rawSenders {
		// TODO: Currently if the validation of the sender fails, it fails silently.
		// This is because the config is loaded before the logging (since the logging
		// depends on the config). But the failure must be registered somehow somewhere.
		if err := sender.Validate(); err == nil {
			validSenders = append(validSenders, sender)
		}
	}
	if validSenders == nil {
		validSenders = append(validSenders, &cfg.Messages.StreamSender)
	}
	cfg.Messages.MainSender = messages.CompositeSender{
		Senders: validSenders,
	}

	return cfg, nil
}
