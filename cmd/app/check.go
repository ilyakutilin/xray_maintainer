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
	serviceName string,
	workDir string,
	executor utils.CommandExecutor,
) error {
	if executor == nil {
		executor = utils.DefaultExecutor
	}

	// TODO: Instead of hardcodig false make it dependable on dryRun
	if err := utils.CheckDirPermissions(workDir, false); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	restartCmd := fmt.Sprintf("sudo systemctl restart %s", serviceName)

	if err := utils.CheckCommandInSudoers(ctx, restartCmd, executor); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	return nil
}
