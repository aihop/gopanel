package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aihop/gopanel/gpc/internal/helper"
	"github.com/aihop/gopanel/gpc/pkg/proto"
	"github.com/aihop/gopanel/gpc/pkg/transport"
	"github.com/spf13/cobra"
)

var goPanelCmd = &cobra.Command{
	Use:          "panel",
	SilenceUsage: true,
}

func init() {
	goPanelCmd.AddCommand(goPanelStatusCmd)
	goPanelCmd.AddCommand(goPanelStartCmd)
	goPanelCmd.AddCommand(goPanelStopCmd)
	goPanelCmd.AddCommand(goPanelRestartCmd)
	goPanelCmd.AddCommand(goPanelInfoCmd)
	goPanelCmd.AddCommand(goPanelUninstallCmd)
	goPanelCmd.AddCommand(goPanelUserInfoCmd)
	goPanelCmd.AddCommand(goPanelResetPasswordCmd)
}

func helperClient() transport.Client {
	return transport.Client{
		SocketPath: cfg.SocketPath,
		Timeout:    actionTimeout(),
	}
}

func callHelper(action string, params map[string]interface{}) (proto.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout()+2*time.Second)
	defer cancel()
	req := proto.Request{
		ID:     fmt.Sprintf("%d", time.Now().UnixNano()),
		Action: action,
		Params: params,
	}
	return helperClient().Do(ctx, req)
}

func handleResponse(resp proto.Response, err error) error {
	if err != nil {
		return err
	}
	if resp.OK {
		return nil
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	if resp.Code != "" {
		return errors.New(resp.Code)
	}
	return errors.New("request failed")
}

var goPanelStatusCmd = &cobra.Command{
	Use:          "status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, err := callHelper("GOPANEL_SERVICE_ACTION", map[string]interface{}{
			"op":   "status",
			"name": cfg.GoPanelServiceName,
		})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		status := resp.Output
		if jsonOut {
			b, _ := json.Marshal(map[string]string{"status": status})
			fmt.Fprintln(os.Stdout, string(b))
			return nil
		}
		fmt.Fprintln(os.Stdout, status)
		return nil
	},
}

var goPanelStartCmd = &cobra.Command{
	Use:          "start",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := callHelper("GOPANEL_SERVICE_ACTION", map[string]interface{}{
			"op":   "start",
			"name": cfg.GoPanelServiceName,
		})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		if resp.Output != "" {
			fmt.Fprintln(os.Stdout, resp.Output)
		}
		return nil
	},
}

var goPanelStopCmd = &cobra.Command{
	Use:          "stop",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := callHelper("GOPANEL_SERVICE_ACTION", map[string]interface{}{
			"op":   "stop",
			"name": cfg.GoPanelServiceName,
		})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		if resp.Output != "" {
			fmt.Fprintln(os.Stdout, resp.Output)
		}
		return nil
	},
}

var goPanelRestartCmd = &cobra.Command{
	Use:          "restart",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := callHelper("GOPANEL_SERVICE_ACTION", map[string]interface{}{
			"op":   "restart",
			"name": cfg.GoPanelServiceName,
		})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		if resp.Output != "" {
			fmt.Fprintln(os.Stdout, resp.Output)
		}
		return nil
	},
}

var goPanelInfoCmd = &cobra.Command{
	Use:          "info",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, err := callHelper("GOPANEL_INFO", map[string]interface{}{
			"name": cfg.GoPanelServiceName,
		})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		if jsonOut {
			fmt.Fprintln(os.Stdout, resp.Output)
			return nil
		}
		fmt.Fprintln(os.Stdout, resp.Output)
		return nil
	},
}

var goPanelUserInfoCmd = &cobra.Command{
	Use:          "user-info",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, err := callHelper("GOPANEL_USER_INFO", map[string]interface{}{})
		if err := handleResponse(resp, err); err != nil {
			return err
		}
		if jsonOut {
			fmt.Fprintln(os.Stdout, resp.Output)
			return nil
		}
		fmt.Fprintln(os.Stdout, resp.Output)
		return nil
	},
}

var goPanelUninstallCmd = &cobra.Command{
	Use:          "uninstall",
	Short:        "Uninstall gopanel and gp-agent artifacts",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := callHelper("GOPANEL_UNINSTALL", map[string]interface{}{})
		if err == nil {
			if err := handleResponse(resp, err); err != nil {
				return err
			}
			if resp.Output != "" {
				fmt.Fprintln(os.Stdout, resp.Output)
			}
			return nil
		}

		// Service socket unavailable — run uninstall directly in this process
		if isSocketError(err) {
			fmt.Fprintln(os.Stderr, "gpc service not running, executing uninstall directly...")
			return runDirectUninstall()
		}
		return err
	},
}

func isSocketError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if os.IsNotExist(err) {
		return true
	}
	return false
}

func runDirectUninstall() error {
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout()+2*time.Second)
	defer cancel()

	hcfg := helper.Config{
		SocketPath:         cfg.SocketPath,
		BaseDir:            cfg.BaseDir,
		GoPanelServiceName: cfg.GoPanelServiceName,
		GoPanelBinaryPath:  cfg.GoPanelBinaryPath,
		GoPanelConfigPath:  cfg.GoPanelConfigPath,
		GoPanelPidfilePath: cfg.GoPanelPidfilePath,
		ActionTimeout:      actionTimeout(),
		LockTimeout:        time.Duration(cfg.LockTimeoutMs) * time.Millisecond,
	}
	out, err := helper.UninstallGoPanel(ctx, hcfg)
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Fprintln(os.Stdout, out)
	}

	// Clean up gpc itself (cannot do this from service mode since gpc.service
	// would kill the handler before the response is sent).
	gpcOut, gpcErr := helper.CleanupGPC(ctx, hcfg)
	if gpcOut != "" {
		fmt.Fprintln(os.Stdout, gpcOut)
	}
	return gpcErr
}

func init() {
	goPanelStatusCmd.Flags().Bool("json", false, "output json")
	goPanelInfoCmd.Flags().Bool("json", false, "output json")
	goPanelUserInfoCmd.Flags().Bool("json", false, "output json")
}
