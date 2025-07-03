package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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
v6: 2001:db8::1
endpoint: engage.cloudflareclient.com:2408`, nil
	}

	countryCode, err := utils.GetCountryCode(ctx)
	if err != nil {
		app.warn(fmt.Sprintf("Failed to get the country that the request for "+
			"the Cloudflare credentials originates from: %v. If such a request hits a "+
			"region block, the request will timed out. This does not prevent further "+
			"execution and the request will be sent anyway.", err))
	}
	if countryCode == "RU" {
		return "", errors.New("the Clouflare credentials generator has been " +
			"launched from Russia. This will inevitably result in the request timeout" +
			"due to a region block, so there is no point in trying. Warp update " +
			"process will now be terminated")
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
		return false, fmt.Errorf("xray verification client failed to start: %w", err)
	}

	app.logger.Info.Println("xray started successfully. Requesting a detailed " +
		"information about the IP address and the provider...")
	proxy := utils.HTTPProxy{IP: "127.0.0.1", Port: xray.Client.Port}
	apiResponse, err := utils.GetRequestWithProxy(ctx, xray.Client.IPCheckerURL, &proxy)

	if err != nil {
		terminateProcess(cmd)
		return false, nil
	}

	app.logger.Info.Println("Response received, shutting down the xray verification " +
		"client...")
	if err := terminateProcess(cmd); err != nil {
		return false, err
	}

	app.logger.Info.Println("Analyzing the response to make sure that the provider " +
		"is Cloudflare...")
	if err := checkIPCheckerResponse(apiResponse, xray.Server.IP); err != nil {
		return false, nil
	}

	return true, nil
}

func updateServerWarpConfig(xrayServerConfig *ServerConfig, cfCreds *CFCreds) error {
	var found bool
	for _, outb := range xrayServerConfig.Outbounds {
		if outb.Protocol == "wireguard" {
			found = true
			outb.Settings.SecretKey = cfCreds.SecretKey
			outb.Settings.Address = []string{cfCreds.V4, cfCreds.V6}
			outb.Settings.Peers = []SrvOutboundSettingsPeer{
				{
					Endpoint:  cfCreds.Endpoint,
					PublicKey: cfCreds.PublicKey,
				},
			}
			outb.Settings.Reserved = cfCreds.Reserved
		}
	}

	if !found {
		return errors.New("wireguard protocol has not been found in the xray server " +
			"config, therefore its update is not possible")
	}

	return nil
}

func (app *Application) updateWarp(ctx context.Context, xray Xray) error {
	app.logger.Info.Println("Checking whether the warp is operational...")

	// Parse the existing xray server config
	app.logger.Info.Println("Parsing the existing xray server config...")
	var xrayServerConfig ServerConfig
	if err := utils.ParseJSONFile(xray.Server.ConfigFilePath, &xrayServerConfig, true); err != nil {
		return fmt.Errorf("error parsing xray server config at path %q: %w", xray.Server.ConfigFilePath, err)
	}
	app.logger.Info.Println("Successfully parsed xray server config.")

	app.logger.Info.Println("Validating xray server config...")
	if err := xrayServerConfig.Validate(); err != nil {
		return fmt.Errorf("the parsed xray server config failed validation: %w", err)
	}
	app.logger.Info.Println("Xray server config successfuly passed validation.")

	// Get the client config and verify that warp is active
	app.logger.Info.Println("Generating a config for the temporary warp verification " +
		"xray client...")
	clientConfig := getClientConfig(&xray.Client, &xray.Server, &xrayServerConfig)
	if err := utils.WriteStructToJSONFile(clientConfig, xray.Client.ConfigFilePath); err != nil {
		return fmt.Errorf("error writing client config to %q: %w", xray.Client.ConfigFilePath, err)
	}
	app.logger.Info.Println("Client config has successfully been generated " +
		"and saved to a file in the main working directory.")
	app.logger.Info.Println("Starting to check if the warp is active and responsive " +
		"using the temporary verification client...")
	warpOK, err := app.isWarpOK(ctx, xray)
	if err != nil {
		return fmt.Errorf("failed to obtain the warp status: %w", err)
	}

	if warpOK {
		app.logger.Info.Println("Warp is active, so its update is not required.")
		return nil
	}

	app.logger.Warning.Println("Warp is not active, so its update is required.")

	app.logger.Info.Println("Launching Cloudflare credential generator to capture " +
		"its output")
	cfCredOutput, err := app.getCFCreds(ctx, xray.CFCredFilePath)
	if err != nil {
		return fmt.Errorf("error while launching the Cloudflare credentials "+
			"generator: %w", err)
	}

	app.logger.Info.Println("Obtained the Cloudflare credentials, parsing...")
	cfCreds, err := parseCFCreds(cfCredOutput)
	if err != nil {
		return fmt.Errorf("error while parsing the generated Cloudflare "+
			"credentials: %w", err)
	}

	app.logger.Info.Println("Successfully parsed the credentials. Updating the xray " +
		"server config with new Warp settings...")
	if err := updateServerWarpConfig(&xrayServerConfig, &cfCreds); err != nil {
		return fmt.Errorf("error updating the xray server config: %w", err)
	}

	app.logger.Info.Println("Writing the new xray server config to file...")
	srvBackupFile, err := utils.BackupFile(xray.Server.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to back up the xray server config file: %w", err)
	}
	if err := utils.WriteStructToJSONFile(&xrayServerConfig, xray.Server.ConfigFilePath); err != nil {
		_ = os.Remove(srvBackupFile)
		return fmt.Errorf("error writing the new xray server config to file: %w", err)
	}

	if !app.debug {
		app.logger.Info.Println("Restarting the xray server service...")
		if err := utils.CheckOperability(ctx, app.xrayServiceName, nil); err != nil {
			app.logger.Info.Println("Xray server service is not operable after " +
				"restart, so reverting the config file to its previous state and " +
				"checking the xray server service operability again...")
			if err := utils.RestoreFile(srvBackupFile, xray.Server.ConfigFilePath); err != nil {
				return fmt.Errorf("error restoring the backup of the xray server "+
					"config file to its original path: %w", err)
			}
			_ = os.Remove(srvBackupFile)
			if err := utils.CheckOperability(ctx, app.xrayServiceName, nil); err != nil {
				return fmt.Errorf("even after restoring the original xray server "+
					"config the service is still inoperable. Further investigation "+
					"is required: %w", err)
			}
		}
		_ = os.Remove(srvBackupFile)
		app.note(fmt.Sprintf("Warp config was corrput. It was updated and now"+
			"the %s is operational with the updated server config.", app.xrayServiceName))
	} else {
		_ = os.Remove(srvBackupFile)
		app.logger.Info.Printf("The app is in debug mode, so the %s will not be restarted.", app.xrayServiceName)
	}

	return nil
}
