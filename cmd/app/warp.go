package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type CFCreds struct {
	SecretKey string
	PublicKey string
	Reserved  []int
	V4        string
	V6        string
	Endpoint  string
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

func getClientConfig(xrayClient *XrayClient, xrayServerConfig *ServerConfig) *ClientConfig {
	var clientConfig ClientConfig

	clientConfig.Log = xrayServerConfig.Log

	clientInbound := ClientInbound{
		Port:     xrayClient.Port,
		Protocol: "http",
	}
	clientConfig.Inbounds = append(clientConfig.Inbounds, clientInbound)

	var cs ClientOutboundSettingsServer

	// Loop through the server inbounds to find the one with the protocol that
	// the warp verification client will use
	// !!! For the moment this works only with shadowsocks !!!
	var found bool
	var routingRuleNetwork string
	for _, inbound := range xrayServerConfig.Inbounds {
		if inbound.Protocol == xrayClient.ServerProtocol {
			found = true
			cs.Port = inbound.Port
			cs.Method = inbound.Settings.Method
			cs.Password = inbound.Settings.Password
			routingRuleNetwork = inbound.Settings.Network
			break
		}
	}

	if !found {
		panic(fmt.Sprintf("protocol %s has not been found in the xray server config "+
			"inbounds, which means that the server config was not properly validated "+
			"after parsing. Check your code so that the protocol required for the "+
			"client operation is supported.", xrayClient.ServerProtocol))
	}

	if cs.Method == "" || cs.Password == "" {
		panic(fmt.Sprintf("protocol %s is present in the xray server config inbounds, "+
			"but it still did not provide the required credentials for the client "+
			"config, which means that the server config was not properly validated "+
			"after parsing. Check your code so that the protocol required for the "+
			"client operation is supported.", xrayClient.ServerProtocol))
	}

	clientOutbound := ClientOutbound{
		Protocol: xrayClient.ServerProtocol,
		Tag:      xrayClient.ServerProtocol,
		Settings: ClientOutboundSettings{
			Servers: []ClientOutboundSettingsServer{cs},
		},
	}
	clientConfig.Outbounds = append(clientConfig.Outbounds, clientOutbound)

	clientRoutingRule := ClientRoutingRule{
		Type:        "field",
		OutboundTag: xrayClient.ServerProtocol,
		Network:     routingRuleNetwork,
	}

	clientRouting := ClientRouting{
		Rules:          []ClientRoutingRule{clientRoutingRule},
		DomainStrategy: "IPIfNonMatch",
	}
	clientConfig.Routing = clientRouting

	return &clientConfig
}

func (app *Application) updateWarp(xray Xray) error {
	app.logger.Info.Println("Updating warp config...")

	// Parse the existing xray config
	app.logger.Info.Println("Parsing the existing xray config...")
	var xrayServerConfig ServerConfig
	if err := utils.ParseJSONFile(xray.Server.ConfigPath, &xrayServerConfig, true); err != nil {
		return fmt.Errorf("error parsing xray server config at path %q: %w", xray.Server.ConfigPath, err)
	}
	app.logger.Info.Println("Successfully parsed xray server config...")

	if err := xrayServerConfig.Validate(); err != nil {
		return fmt.Errorf("the parsed xray server config failed validation: %w", err)
	}

	// Get the client config and verify that warp is active
	clientConfig := getClientConfig(&xray.Client, &xrayServerConfig)

	// TODO: Everything below is temporary for checking
	// You actually need to download the CF cred generator, launch it, parse the output,
	// write new values to the struct, and then write the struct to the json

	fmt.Println(clientConfig)

	for _, outbound := range xrayServerConfig.Outbounds {
		if outbound.Protocol == "wireguard" {
			outbound.Settings.SecretKey = "SOMEVERYSECURESECRETKEY"
		}
	}

	// Open file for writing (or create if it doesn’t exist)
	file, err := os.Create(filepath.Join(filepath.Dir(xray.Server.ConfigPath), "updated.json"))
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
