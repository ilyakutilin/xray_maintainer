package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type ClientInbound struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ClientOutboundSettingsVNextUser struct {
	Id         string `json:"id"`
	Flow       string `json:"flow"`
	Encryption string `json:"encryption"`
}

type ClientOutboundSettingsVNext struct {
	Address string                            `json:"address"`
	Port    int                               `json:"port"`
	Users   []ClientOutboundSettingsVNextUser `json:"users"`
}

type ClientOutboundSettings struct {
	VNext []ClientOutboundSettingsVNext `json:"vnext"`
}

type ClientOutboundStreamRealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
	ShortId     string `json:"shortId"`
	PublicKey   string `json:"publicKey"`
}

type ClientOutboundStreamSettings struct {
	Network         string                              `json:"network"`
	Security        string                              `json:"security"`
	RealitySettings ClientOutboundStreamRealitySettings `json:"realitySettings"`
}

type ClientOutbound struct {
	Protocol       string                       `json:"protocol"`
	Tag            string                       `json:"tag"`
	Settings       ClientOutboundSettings       `json:"settings"`
	StreamSettings ClientOutboundStreamSettings `json:"streamSettings"`
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

func getClientConfig(xrayClient *XrayClient, xrayServerConfig *ServerConfig) *ClientConfig {
	var clientConfig ClientConfig

	clientConfig.Log = Log{Loglevel: "warning"}

	clientInbound := ClientInbound{
		Port:     xrayClient.Port,
		Protocol: "http",
	}
	clientConfig.Inbounds = append(clientConfig.Inbounds, clientInbound)

	// Loop through the server inbounds to find the one with the protocol that
	// the warp verification client will use
	// !!! For the moment this works only with vless !!!
	var inbound *SrvInbound
	for i := range xrayServerConfig.Inbounds {
		if xrayServerConfig.Inbounds[i].Protocol == xrayClient.ServerProtocol {
			inbound = &xrayServerConfig.Inbounds[i] // <-- pointer to actual element
			break
		}
	}

	if inbound == nil {
		panic(fmt.Sprintf("protocol %s has not been found in the xray server config "+
			"inbounds, which means that the server config was not properly validated "+
			"after parsing. Check your code so that the protocol required for the "+
			"client operation is supported.", xrayClient.ServerProtocol))
	}

	server_clients := inbound.Settings.Clients
	var client SrvInbSettingsClient
	if server_clients != nil && len(*server_clients) > 0 {
		client = (*server_clients)[0]
	}
	if client.ID == "" {
		return nil
	}

	clientOutbound := ClientOutbound{
		Protocol: xrayClient.ServerProtocol,
		Tag:      xrayClient.ServerProtocol,
		Settings: ClientOutboundSettings{
			VNext: []ClientOutboundSettingsVNext{
				{
					Address: inbound.Listen,
					Port:    443,
					Users: []ClientOutboundSettingsVNextUser{
						{
							Id:         client.ID,
							Flow:       client.Flow,
							Encryption: inbound.Settings.Decryption,
						},
					},
				},
			},
		},
		StreamSettings: ClientOutboundStreamSettings{
			Network:  inbound.StreamSettings.Network,
			Security: inbound.StreamSettings.Security,
			RealitySettings: ClientOutboundStreamRealitySettings{
				ServerName: inbound.StreamSettings.RealitySettings.ServerNames[0],
				// TODO: Move this to the settings instead of hard coding
				Fingerprint: "edge",
				ShortId:     inbound.StreamSettings.RealitySettings.ShortIds[0],
				PublicKey:   xrayClient.PublicKey,
			},
		},
	}
	clientConfig.Outbounds = append(clientConfig.Outbounds, clientOutbound)

	clientRouting := ClientRouting{
		Rules: []ClientRoutingRule{
			{
				Type:        "field",
				OutboundTag: xrayClient.ServerProtocol,
				Network:     inbound.StreamSettings.Network,
			},
		},
		DomainStrategy: "IPIfNonMatch",
	}
	clientConfig.Routing = clientRouting

	return &clientConfig
}

func startXrayClient(ctx context.Context, xray Xray) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, xray.ExecutableFilePath, "-c", xray.Client.ConfigFilePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start xray process: %w", err)
	}
	return cmd, stdout, nil
}

func watchXrayStartup(stdout io.ReadCloser, ready chan<- struct{}) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("xray:", line)
		if strings.Contains(line, "Failed to start:") {
			close(ready)
			return
		}
		if strings.Contains(line, "Xray ") && strings.Contains(line, " started") {
			close(ready)
			return
		}
	}
}

func waitForXrayReady(ctx context.Context, ready <-chan struct{}, port int) error {
	select {
	case <-ready:
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		return utils.WaitForPort(addr, 3*time.Second)
	case <-ctx.Done():
		return errors.New("timed out waiting for xray startup")
	}
}

func terminateProcess(cmd *exec.Cmd) error {
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("xray process exited with error: %w", err)
	}
	return nil
}
