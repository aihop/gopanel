package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aihop/gopanel/gpc/internal/helper"
	"github.com/spf13/cobra"
)

var helperCmd = &cobra.Command{
	Use:          "helper",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hcfg := helper.Config{
			SocketPath:         cfg.SocketPath,
			BaseDir:            cfg.BaseDir,
			GoPanelServiceName: cfg.GoPanelServiceName,
			GoPanelBinaryPath:  cfg.GoPanelBinaryPath,
			GoPanelConfigPath:  cfg.GoPanelConfigPath,
			GoPanelPidfilePath: cfg.GoPanelPidfilePath,
			ActionTimeout:      actionTimeout(),
			LockTimeout:        time.Duration(cfg.LockTimeoutMs) * time.Millisecond,
			FileRoots:          cfg.FileRoots,
			AllowRootFS:        cfg.AllowRootFS,
			MaxFileReadBytes:   cfg.MaxFileReadBytes,
			MaxFileWriteBytes:  cfg.MaxFileWriteBytes,
		}
		if hcfg.LockTimeout <= 0 {
			hcfg.LockTimeout = 30 * time.Second
		}

		srv := helper.NewServer(hcfg)
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if err := srv.Serve(ctx); err != nil {
			return fmt.Errorf("helper: %w", err)
		}
		return nil
	},
}
