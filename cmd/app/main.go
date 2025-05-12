package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/ilyakutilin/xray_maintainer/messages"
	"github.com/ilyakutilin/xray_maintainer/utils"
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

	// Check if the workdir exists, if not create it
	if err := utils.EnsureDir(cfg.Workdir); err != nil {
		cfg.Messages.MainSender.Send(messages.Message{
			Subject: "Error creating workdir",
			Body:    fmt.Sprintf("Failed to create workdir %s: %v", cfg.Workdir, err),
			Errors:  []error{err},
		})
		app.logger.Error.Fatalf("Error creating workdir: %v", err)
	}

	if err := app.updateMultipleFiles(cfg.Repos, NewFile); err != nil {
		cfg.Messages.MainSender.Send(messages.Message{
			Subject: "Error updating files",
			Body:    fmt.Sprintf("Failed to update the files: %v", err),
			Errors:  []error{err},
		})
		app.logger.Error.Fatalf("Error updating files: %v", err)
	}

	// TODO: Add error handling
	// err = app.updateWarp(cfg.Xray)
	// if err != nil {
	// 	app.logger.Error.Fatalf("Error updating warp config: %v", err)
	// }

	// // TODO: Add error handling
	// _ = RestartService("xray")
	// xrayActive, _ := utils.CheckServiceStatus("xray")
	// if !xrayActive {
	// 	log.Fatal("XRay service failed to start")
	// }

	// TODO: Remove print stmt
	// fmt.Println(cfg.Workdir)
	// fmt.Println(cfg.Xray.Server.IP)
	fmt.Println(cfg)
}
