package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

type Application struct {
	dryRun          bool
	logger          *Logger
	workdir         string
	xrayServiceName string
	notes           []string
	warnings        []string
}

func (app *Application) note(txt string) {
	app.logger.Info.Println(txt)
	app.warnings = append(app.notes, txt)
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
		dryRun:          cfg.DryRun,
		logger:          GetLogger(cfg.DryRun),
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

	ctx := context.Background()

	if err := CheckPermissions(ctx, &app, nil); err != nil {
		app.logger.Error.Fatal(err)
	}

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
		var nw string
		switch {
		case len(app.notes) > 0 && len(app.warnings) == 0:
			nw = "notes"
		case len(app.notes) == 0 && len(app.warnings) > 0:
			nw = "warnings"
		case len(app.notes) > 0 && len(app.warnings) > 0:
			nw = "notes and warnings"
		}
		app.sendMsg(
			cfg.Messages,
			fmt.Sprintf("Completed with %s.", nw),
			fmt.Sprintf("The xray related files and its warp config have been "+
				"successfully checked and updated as necessary, however there are "+
				"some %s:", nw),
		)
	}
}
