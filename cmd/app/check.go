package main

import (
	"context"
	"fmt"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

// CheckPermissions checks the permissions to read / write to the workdir
// and execute the service restart command by the current user
func CheckPermissions(
	ctx context.Context,
	app *Application,
	executor utils.CommandExecutor,
) error {
	if executor == nil {
		executor = utils.DefaultExecutor
	}

	if err := utils.CheckDirPermissions(app.workdir, app.dryRun); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	if !app.dryRun {
		restartCmd := fmt.Sprintf("sudo systemctl restart %s", app.xrayServiceName)

		if err := utils.CheckCommandInSudoers(ctx, restartCmd, executor); err != nil {
			return fmt.Errorf("permission check failed: %w", err)
		}
	}

	return nil
}
