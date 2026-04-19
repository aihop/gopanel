package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/aihop/gopanel/gp-agent/global"
	"github.com/aihop/gopanel/gp-agent/init/caddy"
	"github.com/aihop/gopanel/gp-agent/init/daemon"
	"github.com/aihop/gopanel/gp-agent/pkg/transport"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serveCmd = &cobra.Command{
	Use:          "serve",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		socketPath := global.CONF.SocketPath
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		enableCaddy, _ := rootCmd.PersistentFlags().GetBool("enable-caddy")
		if enableCaddy {
			if err := caddy.Init(); err != nil {
				return fmt.Errorf("caddy init: %w", err)
			}
		}

		enableDaemon, _ := rootCmd.PersistentFlags().GetBool("enable-daemon")
		if enableDaemon {
			if err := daemon.Init(); err != nil {
				return fmt.Errorf("daemon init: %w", err)
			}
		}

		if global.LOG != nil {
			global.LOG.Info("gp-agent serving", zap.String("socket", socketPath), zap.String("os", runtime.GOOS))
		}
		if err := transport.Serve(ctx, socketPath); err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	},
}
