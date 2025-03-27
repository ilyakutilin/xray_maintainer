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
}

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

	// TODO: Check that the workdir exists, and if not, create it

	xrayExecutable := File{
		filePath:    filepath.Join(cfg.workdirPath, "xray"),
		releaseURL:  cfg.xrayCoreReleaseInfoURL,
		downloadURL: cfg.xrayCoreDownloadURL,
	}
	// TODO: Add error handling
	_ = updateFile(xrayExecutable, cfg.debug)

	// geoipFile := File{
	// 	filePath:    filepath.Join(cfg.workdirPath, "geoip.dat"),
	// 	releaseURL:  cfg.geoipReleaseInfoURL,
	// 	downloadURL: cfg.geoipDownloadURL,
	// }
	// // TODO: Add error handling
	// _ = updateFile(geoipFile, cfg.debug)

	geositeFile := File{
		filePath:    filepath.Join(cfg.workdirPath, "geosite.dat"),
		releaseURL:  cfg.geositeReleaseInfoURL,
		downloadURL: cfg.geositeDownloadURL,
	}
	// TODO: Add error handling
	_ = updateFile(geositeFile, cfg.debug)

	// TODO: Add error handling
	err = updateWarp(cfg.warp, cfg.debug)
	if err != nil {
		logger.Error.Fatalf("Error updating warp config: %v", err)
	}

	// // TODO: Add error handling
	// _ = restartService("xray")
	// xrayActive, _ := checkServiceStatus("xray")
	// if !xrayActive {
	// 	log.Fatal("XRay service failed to start")
	// }

	// TODO: Remove print stmt
	fmt.Println(cfg.workdirPath)
	fmt.Println(cfg.warp.xrayServerIP)
	fmt.Println(xrayExecutable)
}
