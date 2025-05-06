package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	// "github.com/ilyakutilin/xray_maintainer/utils"
)

type Application struct {
	debug   bool
	logger  *Logger
	workdir string
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	app := Application{
		debug:   cfg.Debug,
		logger:  GetLogger(cfg.Debug),
		workdir: cfg.Workdir,
	}

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			app.logger.Error.Printf("PANIC: %v\n%s", r, stack)
			// TODO: Send a message with the panic info
			os.Exit(1)
		}
	}()

	// TODO: Check that the workdir exists, and if not, create it

	xrayExecutable := NewFile(cfg.Repos.XrayCore)
	// TODO: Add error handling
	_ = app.updateFile(xrayExecutable, cfg.Debug)

	geoipFile := NewFile(cfg.Repos.Geoip)
	// TODO: Add error handling
	_ = app.updateFile(geoipFile, cfg.Debug)

	geositeFile := NewFile(cfg.Repos.Geosite)
	// TODO: Add error handling
	_ = app.updateFile(geositeFile, cfg.Debug)

	// TODO: Add error handling
	err = app.updateWarp(cfg.Xray)
	if err != nil {
		app.logger.Error.Fatalf("Error updating warp config: %v", err)
	}

	// // TODO: Add error handling
	// _ = RestartService("xray")
	// xrayActive, _ := utils.CheckServiceStatus("xray")
	// if !xrayActive {
	// 	log.Fatal("XRay service failed to start")
	// }

	// TODO: Remove print stmt
	fmt.Println(cfg.Workdir)
	fmt.Println(cfg.Xray.Server.IP)
	fmt.Println(xrayExecutable)
}
