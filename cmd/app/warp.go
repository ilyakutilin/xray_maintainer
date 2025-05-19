package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

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

func (app Application) getCFCreds(ctx context.Context, cfCredFilePath string) (string, error) {
	if app.debug {
		return `device_id: abcdefab-0123-01ab-23cd-0123abcd4567
token: deadbeef-0000-cafe-babe-0000feedface
account_id: abcdef12-3456-aaaa-bbbb-cccc12345678
account_type: free
license: ExamplE1-Fake1234-DemoTest
private_key: FAKEFAKE/DEMO1234+NOTREALDATA==EXAMPLE12/==
public_key: ZZZZ0000/FAKEYFAK+12345678/DEMODEMO==TEST
client_id: a1o2
reserved: [ 100, 200, 300 ]
v4: 172.16.0.2
v6: dead:beef:0000:fake:1234:cafe:feed:0001
endpoint: engage.cloudflareclient.com:2408`, nil
	}

	return utils.ExecuteCommand(ctx, cfCredFilePath)
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

func getClientConfig(xrayClient *XrayClient, xrayServer *XrayServer, xrayServerConfig *ServerConfig) *ClientConfig {
	var clientConfig ClientConfig

	clientConfig.Log = Log{Loglevel: "warning"}

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
			cs.Address = xrayServer.IP
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

func checkIPCheckerResponse(ipCheckerResponseJSON []byte, xrayServerIP string) error {
	type IPCheckerResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		ISP     string `json:"isp"`
		Org     string `json:"org"`
		Query   string `json:"query"`
	}

	var r IPCheckerResponse

	if err := utils.ParseJSON(ipCheckerResponseJSON, &r, false); err != nil {
		return fmt.Errorf("could not parse the JSON response provided by the ip "+
			"checker into a struct: %w", err)
	}

	if r.Status != "success" {
		return fmt.Errorf("the ip checker failed to get the ISP/Org status for IP %s "+
			"and provided the following message: %s", r.Query, r.Message)
	}

	if r.Query == xrayServerIP {
		return fmt.Errorf("ip address detected by the ip checker is %s which is "+
			"the address of the xray server machine, which means that Warp is not "+
			"active", r.Query)
	}

	if !strings.Contains(strings.ToLower(r.ISP), "cloudflare") || !strings.Contains(strings.ToLower(r.Org), "cloudflare") {
		return fmt.Errorf("ip checker could not detect Cloudflare in ISP or Org "+
			"which means that Warp is not active. ISP: %s; Org: %s. Full response: %v", r.ISP, r.Org, r)
	}

	return nil
}

func (app *Application) isWarpOK(ctx context.Context, xray Xray) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd, stdout, err := startXrayClient(ctx, xray)
	if err != nil {
		return false, err
	}

	ready := make(chan struct{})
	go watchXrayStartup(stdout, ready)

	if err := waitForXrayReady(ctx, ready, xray.Client.Port); err != nil {
		terminateProcess(cmd)
		return false, err
	}

	app.logger.Info.Println("xray started successfully. Performing IP info request...")
	proxy := utils.HTTPProxy{IP: "127.0.0.1", Port: xray.Client.Port}
	apiResponse, err := utils.GetRequestWithProxy(ctx, xray.Client.IPCheckerURL, &proxy)

	if err != nil {
		terminateProcess(cmd)
		return false, fmt.Errorf("failed to fetch the warp status JSON from the ip checker API: %w", err)
	}

	app.logger.Info.Println("sending SIGTERM to the xray verification client...")
	if err := terminateProcess(cmd); err != nil {
		return false, err
	}

	if err := checkIPCheckerResponse(apiResponse, xray.Server.IP); err != nil {
		return false, nil
	}

	return true, nil
}

func (app *Application) updateWarp(ctx context.Context, xray Xray) error {
	app.logger.Info.Println("Updating warp config...")

	// Parse the existing xray server config
	app.logger.Info.Println("Parsing the existing xray server config...")
	var xrayServerConfig ServerConfig
	if err := utils.ParseJSONFile(xray.Server.ConfigFilePath, &xrayServerConfig, true); err != nil {
		return fmt.Errorf("error parsing xray server config at path %q: %w", xray.Server.ConfigFilePath, err)
	}
	app.logger.Info.Println("Successfully parsed xray server config.")

	if err := xrayServerConfig.Validate(); err != nil {
		return fmt.Errorf("the parsed xray server config failed validation: %w", err)
	}

	// Get the client config and verify that warp is active
	clientConfig := getClientConfig(&xray.Client, &xray.Server, &xrayServerConfig)
	if err := utils.WriteStructToJSONFile(clientConfig, xray.Client.ConfigFilePath); err != nil {
		return fmt.Errorf("error writing client config to %q: %w", xray.Client.ConfigFilePath, err)
	}
	app.logger.Info.Println("Successfully wrote client config to file.")
	app.logger.Info.Println("Starting to check if the warp is active and responsive...")
	warpOK, err := app.isWarpOK(ctx, xray)
	if err != nil {
		return fmt.Errorf("failed to obtain the warp status: %w", err)
	}

	// TODO: Remove this line
	warpOK = !warpOK

	if warpOK {
		app.logger.Info.Println("Warp is active, so its update is not required.")
		return nil
	}

	app.logger.Error.Println("Warp is not active, so its update is required.")

	// TODO: Launch the CF cred generator, parse the output,
	// write new values to the struct, and then write the struct to the json

	// Launch Cloudflare credential generator and capture its output
	// TODO: At this point it just freezes. Check how the generator launches and what it does.
	cfCredOutput, err := app.getCFCreds(ctx, xray.CFCredFilePath)
	if err != nil {
		return fmt.Errorf("error while launching the Cloudflare credentials "+
			"generator: %w", err)
	}

	// Parse the Cloudflare credential generator output into a struct
	cfCreds, err := parseCFCreds(cfCredOutput)
	if err != nil {
		return fmt.Errorf("error while parsing the generated Cloudflare "+
			"credentials: %w", err)
	}

	// Update the xray server config with new Warp settings
	fmt.Printf(
		"CFCreds struct:\n"+
			"SecretKey: %s\n"+
			"PublicKey: %s\n"+
			"Reserved: %v\n"+
			"V4: %s\n"+
			"V6: %s\n"+
			"Endpoint: %s\n",
		cfCreds.SecretKey,
		cfCreds.PublicKey,
		cfCreds.Reserved,
		cfCreds.V4,
		cfCreds.V6,
		cfCreds.Endpoint,
	)

	return nil
}
