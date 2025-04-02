package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CFCreds struct {
	SecretKey string
	PublicKey string
	Reserved  []int
	V4        string
	V6        string
	Endpoint  string
}

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

// Parses the Cloudflare generator output. Tailored specifically for the output of
// github.com/badafans/warp-reg.
func parseCFCreds(output string) (CFCreds, error) {
	var result CFCreds

	patterns := map[string]*regexp.Regexp{
		"private_key": regexp.MustCompile(`(?m)^private_key:\s*(\S+)`),
		"public_key":  regexp.MustCompile(`(?m)^public_key:\s*(\S+)`),
		"reserved":    regexp.MustCompile(`(?m)^reserved:\s*\[([0-9,\s]+)\]`),
		"v4":          regexp.MustCompile(`(?m)^v4:\s*(\S+)`),
		"v6":          regexp.MustCompile(`(?m)^v6:\s*(\S+)`),
		"endpoint":    regexp.MustCompile(`(?m)^endpoint:\s*(\S+)`),
	}

	for key, pattern := range patterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) < 2 {
			return result, errors.New("missing required field: " + key)
		}
		switch key {
		case "private_key":
			result.SecretKey = matches[1]
		case "public_key":
			result.PublicKey = matches[1]
		case "reserved":
			values := strings.Split(matches[1], ",")
			for _, v := range values {
				var num int
				fmt.Sscanf(strings.TrimSpace(v), "%d", &num)
				result.Reserved = append(result.Reserved, num)
			}
		case "v4":
			result.V4 = matches[1]
		case "v6":
			result.V6 = matches[1]
		case "endpoint":
			result.Endpoint = matches[1]
		}
	}

	return result, nil
}

// parseJSONFile reads a JSON file and decodes it into the given target.
// target must be a non-nil pointer to a struct/map/slice that matches the JSON structure.
// If strict is true, unknown fields in the JSON file will result in an error.
// Returns an error if file reading or JSON parsing fails.
func parseJSONFile[T any](jsonFilePath string, target *T, strict bool) error {
	if target == nil {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	if !fileExists(jsonFilePath) {
		return fmt.Errorf("file %q does not exist", filepath.Base(jsonFilePath))
	}

	file, err := os.Open(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to open JSON file %q: %w", jsonFilePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	if strict {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON from %q: %w", jsonFilePath, err)
	}

	return nil
}

func updateWarp(warpConfig Warp, debug bool) error {
	logger := GetLogger(debug)

	logger.Info.Println("Updating warp config...")
	logger.Info.Println("Parsing the existing xray config...")
	xrayServerConfig := ServerConfig{}
	if err := parseJSONFile(warpConfig.xrayServerConfigPath, &xrayServerConfig, true); err != nil {
		return fmt.Errorf("error parsing xray server config at path %q: %w", warpConfig.xrayServerConfigPath, err)
	}
	logger.Info.Println("Successfully parsed xray server config...")

	// TODO: Everything below is temporary for checking
	// You actually need to download the CF cred generator, launch it, parse the output,
	// write new values to the struct, and then write the struct to the json

	for _, outbound := range xrayServerConfig.Outbounds {
		if outbound.Protocol == "wireguard" {
			outbound.Settings.SecretKey = "SOMEVERYSECURESECRETKEY"
		}
	}

	// Open file for writing (or create if it doesn’t exist)
	file, err := os.Create(filepath.Join(filepath.Dir(warpConfig.xrayServerConfigPath), "updated.json"))
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Create a JSON encoder and write to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	if err := encoder.Encode(xrayServerConfig); err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	return nil
}
