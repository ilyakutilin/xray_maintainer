package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type Application struct {
	debug           bool
	logger          *Logger
	workdir         string
	xrayServiceName string
	notes           []string
	warnings        []string
}

func (app *Application) warn(txt string) {
	app.logger.Warning.Println(txt)
	app.warnings = append(app.warnings, txt)
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	app := Application{
		debug:           cfg.Debug,
		logger:          GetLogger(cfg.Debug),
		workdir:         cfg.Workdir,
		xrayServiceName: cfg.Xray.Server.ServiceName,
	}

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			app.sendMsg(
				cfg.Messages,
				"App panicked",
				fmt.Sprintf("Panic in the app:\n%v\n%s", r, stack),
			)
			app.logger.Error.Printf("PANIC: %v\n%s", r, stack)
			os.Exit(1)
		}
	}()

	// Check if the workdir exists, if not create it
	if err := utils.EnsureDir(cfg.Workdir); err != nil {
		app.sendMsg(
			cfg.Messages,
			"Error creating workdir",
			fmt.Sprintf("Failed to create the main app workdir %s "+
				"due to the following error:\n%v\nThe process stopped at this point "+
				"and nothing else was done.", cfg.Workdir, err),
		)
		app.logger.Error.Fatalf("Error creating workdir: %v", err)
	}

	ctx := context.Background()

	if err := app.updateMultipleFiles(ctx, cfg.Repos, NewFile); err != nil {
		app.sendMsg(
			cfg.Messages,
			"Error updating files",
			fmt.Sprintf("Failed to update the files: %v", err),
		)
		app.logger.Error.Fatalf("Error updating files: %v", err)
	}

	err = app.updateWarp(ctx, cfg.Xray)
	if err != nil {
		app.sendMsg(
			cfg.Messages,
			"Error updating the warp config",
			fmt.Sprintf("Failed to update the warp config: %v", err),
		)
		app.logger.Error.Fatalf("Error updating warp config: %v", err)
	}

	if len(app.notes) > 0 || len(app.warnings) > 0 {
		app.sendMsg(
			cfg.Messages,
			"Success with notes and/or warnings.",
			"The xray related files and its warp config have been successfully "+
				"checked and updated as necessary, however there are some notes "+
				"and/or warnings:",
		)
	}
}
