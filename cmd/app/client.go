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
