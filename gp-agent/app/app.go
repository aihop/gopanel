package app

import (
	"fmt"
	"os"

	"github.com/aihop/gopanel/gp-agent/app/cmd"
)

func Run() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "service")
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
