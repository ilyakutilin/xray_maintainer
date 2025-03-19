package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/joho/godotenv"
)

type File struct {
	filePath    string
	releaseURL  string
	downloadURL string
	executable  bool
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Could not load .env file - relying on flags and defaults...")
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	xrayExecutable := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "xray"),
		releaseURL:  cfg.xrayCoreReleaseInfoURL,
		downloadURL: cfg.xrayCoreDownloadURL,
		executable:  true,
	}
	// TODO: Add error handling
	xrayExecutable, _ = updateFile(xrayExecutable)

	geoipFile := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "geoip.dat"),
		releaseURL:  cfg.geoipReleaseInfoURL,
		downloadURL: cfg.geoipDownloadURL,
		executable:  false,
	}
	// TODO: Add error handling
	geoipFile, _ = updateFile(geoipFile)

	geositeFile := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "geosite.dat"),
		releaseURL:  cfg.geositeReleaseInfoURL,
		downloadURL: cfg.geositeDownloadURL,
		executable:  false,
	}
	// TODO: Add error handling
	geositeFile, _ = updateFile(geositeFile)

	// TODO: Add error handling
	_ = updateWarp(cfg.xrayServerIP, cfg.xrayProtocol, cfg.xrayClientPort, cfg.ipCheckerURL, cfg.cfCredGenURL)

	// TODO: Add error handling
	_ = restartService("xray")
	xrayActive, _ := checkServiceStatus("xray")
	if !xrayActive {
		log.Fatal("XRay service failed to start")
	}

	// TODO: Remove print stmt
	fmt.Println(cfg.xrayDirPath)
	fmt.Println(cfg.xrayServerIP)
	fmt.Println(xrayExecutable)
	fmt.Println(geoipFile)
	fmt.Println(geositeFile)
}
