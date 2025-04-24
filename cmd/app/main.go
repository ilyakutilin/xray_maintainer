package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/joho/godotenv"
	// "github.com/ilyakutilin/xray_maintainer/utils"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Could not load .env file - relying on flags and defaults...")
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	logger := GetLogger(cfg.debug)

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			logger.Error.Printf("PANIC: %v\n%s", r, stack)
			// TODO: Send a message with the panic info
			os.Exit(1)
		}
	}()

	// TODO: Check that the workdir exists, and if not, create it

	xrayExecutable := NewFile(
		filepath.Join(cfg.workdirPath, "xray"),
		cfg.xrayCoreReleaseInfoURL,
		cfg.xrayCoreDownloadURL,
	)
	// TODO: Add error handling
	_ = updateFile(xrayExecutable, cfg.debug)

	// geoipFile := NewFile(
	// 	filepath.Join(cfg.workdirPath, "geoip.dat"),
	// 	cfg.geoipReleaseInfoURL,
	// 	cfg.geoipDownloadURL,
	// )
	// // TODO: Add error handling
	// _ = updateFile(geoipFile, cfg.debug)

	geositeFile := NewFile(
		filepath.Join(cfg.workdirPath, "geosite.dat"),
		cfg.geositeReleaseInfoURL,
		cfg.geositeDownloadURL,
	)
	// TODO: Add error handling
	_ = updateFile(geositeFile, cfg.debug)

	// TODO: Add error handling
	err = updateWarp(cfg.warp, cfg.debug)
	if err != nil {
		logger.Error.Fatalf("Error updating warp config: %v", err)
	}

	// // TODO: Add error handling
	// _ = RestartService("xray")
	// xrayActive, _ := utils.CheckServiceStatus("xray")
	// if !xrayActive {
	// 	log.Fatal("XRay service failed to start")
	// }

	// TODO: Remove print stmt
	fmt.Println(cfg.workdirPath)
	fmt.Println(cfg.warp.xrayServerIP)
	fmt.Println(xrayExecutable)
}
