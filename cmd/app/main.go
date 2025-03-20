package main

import (
	"fmt"
	"path/filepath"

	"github.com/joho/godotenv"
)

type File struct {
	filePath    string
	releaseURL  string
	downloadURL string
}

func main() {
	logger := GetLogger()

	err := godotenv.Load()
	if err != nil {
		logger.Info.Println("Could not load .env file - relying on flags and defaults...")
	}

	cfg, err := loadConfig()
	if err != nil {
		logger.Error.Fatalf("Error loading config: %v", err)
	}

	xrayExecutable := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "xray"),
		releaseURL:  cfg.xrayCoreReleaseInfoURL,
		downloadURL: cfg.xrayCoreDownloadURL,
	}
	// TODO: Add error handling
	_ = updateFile(xrayExecutable)

	geoipFile := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "geoip.dat"),
		releaseURL:  cfg.geoipReleaseInfoURL,
		downloadURL: cfg.geoipDownloadURL,
	}
	// TODO: Add error handling
	_ = updateFile(geoipFile)

	geositeFile := File{
		filePath:    filepath.Join(cfg.xrayDirPath, "geosite.dat"),
		releaseURL:  cfg.geositeReleaseInfoURL,
		downloadURL: cfg.geositeDownloadURL,
	}
	// TODO: Add error handling
	_ = updateFile(geositeFile)

	// // TODO: Add error handling
	// _ = updateWarp(cfg.xrayServerIP, cfg.xrayProtocol, cfg.xrayClientPort, cfg.ipCheckerURL, cfg.cfCredGenURL)

	// // TODO: Add error handling
	// _ = restartService("xray")
	// xrayActive, _ := checkServiceStatus("xray")
	// if !xrayActive {
	// 	log.Fatal("XRay service failed to start")
	// }

	// TODO: Remove print stmt
	fmt.Println(cfg.xrayDirPath)
	fmt.Println(cfg.xrayServerIP)
	fmt.Println(xrayExecutable)
	fmt.Println(geoipFile)
	fmt.Println(geositeFile)
}
