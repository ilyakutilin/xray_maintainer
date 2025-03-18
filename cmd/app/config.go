package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Holds the application configuration
type Config struct {
	xrayServerIP          string
	xrayServerPath        string
	xrayProtocol          string
	xrayClientPort        int
	ipCheckerURL          string
	cfCredGenPath         string
	cfCredGenURL          string
	geoipReleaseInfoURL   string
	geoipDownloadURL      string
	geositeReleaseInfoURL string
	geositeDownloadURL    string
	xrayCoreDownloadURL   string
}

// Loads configuration from flags, environment variables, or defaults
func LoadConfig() (*Config, error) {
	// Define CLI flags (there are no defaults for CLI flags since defaults are handled
	// by environment variables)
	xrayServerIPFlag := flag.String("xray_server_ip", "", "XRay server IP")
	xrayServerPathFlag := flag.String("xray_server_path", "", "Path of the xray server directory with the executable, config and geofiles")
	xrayProtocolFlag := flag.String("xray_protocol", "", "XRay protocol")
	xrayClientPortFlag := flag.Int("xray_client_port", 0, "Port used by the test throwaway xray client")
	ipCheckerURLFlag := flag.String("ip_checker_url", "", "IP checker service URL")
	cfCredGenPathFlag := flag.String("cf_gen_path", "", "Path for the Cloudflare credential generator")
	cfCredGenURLFlag := flag.String("cf_gen_url", "", "URL for downloading the Cloudflare credential generator")
	geoipReleaseInfoURLFlag := flag.String("geoip_rel_url", "", "URL for fetching geoip release info")
	geoipDownloadURLFlag := flag.String("geoip_dl_url", "", "URL for downloading a geoip.dat file")
	geositeReleaseInfoURLFlag := flag.String("geosite_rel_url", "", "URL for fetching geosite release info")
	geositeDownloadURLFlag := flag.String("geosite_url", "", "URL for downloading a geosite.dat file")
	xrayCoreDownloadURLFlag := flag.String("xray_core_dl_url", "", "XRay server executable URL")

	flag.Parse()

	cfg := &Config{}

	var err error

	cfg.xrayServerIP, err = getPriorityString(*xrayServerIPFlag, "XRAY_SERVER_IP", "")
	if err != nil {
		return nil, err
	}
	cfg.xrayServerPath, _ = getPriorityString(*xrayServerPathFlag, "XRAY_SERVER_PATH", "/opt/xray/")
	cfg.xrayProtocol, _ = getPriorityString(*xrayProtocolFlag, "XRAY_PROTOCOL", "shadowsocks")
	cfg.xrayClientPort, err = getPriorityInt(*xrayClientPortFlag, "XRAY_CLIENT_PORT", 10801)
	if err != nil {
		return nil, err
	}
	cfg.ipCheckerURL, _ = getPriorityString(*ipCheckerURLFlag, "IP_CHECKER_URL", "http://ip-api.com/json/")
	cfg.cfCredGenPath, _ = getPriorityString(*cfCredGenPathFlag, "CF_CRED_GEN_PATH", "/opt/xray/")
	cfg.cfCredGenURL, _ = getPriorityString(*cfCredGenURLFlag, "CF_CRED_GEN_URL", "https://github.com/badafans/warp-reg/releases/download/latest/main-linux-amd64")
	cfg.geoipReleaseInfoURL, _ = getPriorityString(*geoipReleaseInfoURLFlag, "GEOIP_RELEASE_INFO_URL", "https://api.github.com/repos/v2fly/geoip/releases/latest")
	cfg.geoipDownloadURL, _ = getPriorityString(*geoipDownloadURLFlag, "GEOIP_DOWNLOAD_URL", "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat")
	cfg.geositeReleaseInfoURL, _ = getPriorityString(*geositeReleaseInfoURLFlag, "GEOSITE_RELEASE_INFO_URL", "https://api.github.com/repos/v2fly/domain-list-community/releases/latest")
	cfg.geositeDownloadURL, _ = getPriorityString(*geositeDownloadURLFlag, "GEOSITE_DOWNLOAD_URL", "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat")
	cfg.xrayCoreDownloadURL, _ = getPriorityString(*xrayCoreDownloadURLFlag, "XRAY_CORE_DOWNLOAD_URL", "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip")

	return cfg, nil
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
// func getPriorityBool(flagValue bool, envKey string, defaultValue bool) bool {
// 	if flagValue {
// 		return flagValue
// 	}
// 	if value, exists := os.LookupEnv(envKey); exists {
// 		if boolValue, err := strconv.ParseBool(value); err == nil {
// 			return boolValue
// 		}
// 	}
// 	return defaultValue
// }
