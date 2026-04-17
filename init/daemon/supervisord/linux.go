//go:build !windows
// +build !windows

package supervisord

import (
	"fmt"
	"os"

	log "github.com/aihop/gopanel/pkg/zlog"
	"github.com/jessevdk/go-flags"
	"github.com/ochinchina/go-reaper"
)

// ReapZombie 在 Unix-like 平台启动 reaper goroutine，回收子进程
func ReapZombie() {
	go reaper.Reap()
}

func Init() {
	if BuildVersion != "" {
		VERSION = BuildVersion
	}
	ReapZombie()

	// when execute `supervisord` without sub-command, it should start the server
	parser.Command.SubcommandsOptional = true
	parser.CommandHandler = func(command flags.Commander, args []string) error {
		if command == nil {
			log.SetOutput(os.Stdout)
			if options.Daemon {
				logFile := getSupervisordLogFile(options.Configuration)
				Daemonize(logFile, runServer)
			} else {
				runServer()
			}
			os.Exit(0)
		}
		return command.Execute(args)
	}

	if _, err := parser.Parse(); err != nil {
		flagsErr, ok := err.(*flags.Error)
		if ok {
			switch flagsErr.Type {
			case flags.ErrHelp:
				_, _ = fmt.Fprintln(os.Stdout, err)
				os.Exit(0)
			default:
				_, _ = fmt.Fprintf(os.Stderr, "error when parsing command: %s\n", err)
				os.Exit(1)
			}
		}
	}
}
