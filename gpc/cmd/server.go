package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:          "server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd
		_ = args
		return errors.New("gpc server is optional; keep server responsibilities in GoPanel")
	},
}
