package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Holds the settings required for Cloudflare Warp
type Warp struct {
	xrayServerConfigPath string
	xrayServerIP         string
	xrayProtocol         string
	xrayClientPort       int
	ipCheckerURL         string
	cfCredGenPath        string
	cfCredGenURL         string
}

// Holds the application configuration
type Config struct {
	debug                  bool
	workdirPath            string
	geoipReleaseInfoURL    string
	geoipDownloadURL       string
	geositeReleaseInfoURL  string
	geositeDownloadURL     string
	xrayCoreReleaseInfoURL string
	xrayCoreDownloadURL    string
	warp                   Warp
}

type AllowedTypes interface {
	string | int | bool
}

func execFn[T AllowedTypes](fn func(T, string, T) (T, error), flagValue T, envKey string, defaultValue T, err *error) T {
	var value T

	if *err != nil {
		return value
	}

	value, *err = fn(flagValue, envKey, defaultValue)
	return value
}

// Gets the string value from flag > env var > default
func getPriorityString(flagValue string, envKey string, defaultValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if value, exists := os.LookupEnv(envKey); exists {
		return value, nil
	}
	if defaultValue == "" {
		return "", fmt.Errorf("%s environment variable is required", envKey)
	}
	return defaultValue, nil
}

// Gets the int value from flag > env var > default
func getPriorityInt(flagValue int, envKey string, defaultValue int) (int, error) {
	if flagValue != 0 {
		return flagValue, nil
	}
	if value, exists := os.LookupEnv(envKey); exists {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}
		return intValue, nil
	}
	if defaultValue == 0 {
		return 0, fmt.Errorf("error: %s environment variable is required", envKey)
	}
	return defaultValue, nil
}

// Gets the value from flag > env var > default
func getPriorityBool(flagValue bool, envKey string, defaultValue bool) bool {
	if flagValue {
		return flagValue
	}
	if value, exists := os.LookupEnv(envKey); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Loads configuration from flags, environment variables, or defaults
func loadConfig() (*Config, error) {
	// Define CLI flags (there are no defaults for CLI flags since defaults are handled
	// by environment variables)
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	workdirPathFlag := flag.String("xray-dir-path", "", "Path of the xray server directory with the executable, config and geofiles")
	xrayServerConfigFileNameFlag := flag.String("xray-server-config-filename", "", "Name of the XRay server config file")
	xrayServerIPFlag := flag.String("xray-server-ip", "", "XRay server IP")
	xrayProtocolFlag := flag.String("xray-protocol", "", "XRay protocol")
	xrayClientPortFlag := flag.Int("xray-client-port", 0, "Port used by the test throwaway xray client")
	ipCheckerURLFlag := flag.String("ip-checker-url", "", "IP checker service URL")
	cfCredGenFileNameFlag := flag.String("cf-cred-gen-filename", "", "Name of the Cloudflare credential generator executable")
	cfCredGenURLFlag := flag.String("cf-gen-url", "", "URL for downloading the Cloudflare credential generator")
	geoipReleaseInfoURLFlag := flag.String("geoip-rel-url", "", "URL for fetching geoip release info")
	geoipDownloadURLFlag := flag.String("geoip-dl-url", "", "URL for downloading a geoip.dat file")
	geositeReleaseInfoURLFlag := flag.String("geosite-rel-url", "", "URL for fetching geosite release info")
	geositeDownloadURLFlag := flag.String("geosite-url", "", "URL for downloading a geosite.dat file")
	xrayCoreReleaseInfoURLFlag := flag.String("xray-core-rel-url", "", "URL for fetching the XRay server executable release info")
	xrayCoreDownloadURLFlag := flag.String("xray-core-dl-url", "", "XRay server executable URL")

	flag.Parse()

	cfg := &Config{}

	cfg.debug = getPriorityBool(*debugFlag, "DEBUG", false)

	var err error

	workdirPath := execFn(getPriorityString, *workdirPathFlag, "WORKDIR_PATH", "/opt/xray", &err)
	xrayServerConfigFileName := execFn(getPriorityString, *xrayServerConfigFileNameFlag, "XRAY_SERVER_CONFIG_FILENAME", "config.json", &err)
	cfCredGenFileName := execFn(getPriorityString, *cfCredGenFileNameFlag, "CF_CRED_GEN_FILENAME", "main-linux-amd64", &err)
	cfg.warp.xrayServerConfigPath = filepath.Join(workdirPath, "config.json")
	cfg.warp.xrayServerIP = execFn(getPriorityString, *xrayServerIPFlag, "XRAY_SERVER_IP", "", &err)
	cfg.warp.xrayProtocol = execFn(getPriorityString, *xrayProtocolFlag, "XRAY_PROTOCOL", "shadowsocks", &err)
	cfg.warp.xrayClientPort = execFn(getPriorityInt, *xrayClientPortFlag, "XRAY_CLIENT_PORT", 10801, &err)
	cfg.warp.ipCheckerURL = execFn(getPriorityString, *ipCheckerURLFlag, "IP_CHECKER_URL", "http://ip-api.com/json/", &err)
	cfg.warp.cfCredGenURL = execFn(getPriorityString, *cfCredGenURLFlag, "CF_CRED_GEN_URL", "https://github.com/badafans/warp-reg/releases/download/latest/main-linux-amd64", &err)
	cfg.geoipReleaseInfoURL = execFn(getPriorityString, *geoipReleaseInfoURLFlag, "GEOIP_RELEASE_INFO_URL", "https://api.github.com/repos/v2fly/geoip/releases/latest", &err)
	cfg.geoipDownloadURL = execFn(getPriorityString, *geoipDownloadURLFlag, "GEOIP_DOWNLOAD_URL", "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat", &err)
	cfg.geositeReleaseInfoURL = execFn(getPriorityString, *geositeReleaseInfoURLFlag, "GEOSITE_RELEASE_INFO_URL", "https://api.github.com/repos/v2fly/domain-list-community/releases/latest", &err)
	cfg.geositeDownloadURL = execFn(getPriorityString, *geositeDownloadURLFlag, "GEOSITE_DOWNLOAD_URL", "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat", &err)
	cfg.xrayCoreReleaseInfoURL = execFn(getPriorityString, *xrayCoreReleaseInfoURLFlag, "XRAY_CORE_RELEASE_INFO_URL", "https://api.github.com/repos/XTLS/Xray-core/releases/latest", &err)
	cfg.xrayCoreDownloadURL = execFn(getPriorityString, *xrayCoreDownloadURLFlag, "XRAY_CORE_DOWNLOAD_URL", "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip", &err)

	if err != nil {
		return nil, err
	}

	cfg.workdirPath, err = expandPath(workdirPath)
	if err != nil {
		return nil, err
	}

	cfg.warp.xrayServerConfigPath = filepath.Join(cfg.workdirPath, xrayServerConfigFileName)
	cfg.warp.cfCredGenPath = filepath.Join(cfg.workdirPath, cfCredGenFileName)

	return cfg, nil
}
