//go:build !windows
// +build !windows

package supervisord

import (
	log "github.com/aihop/gopanel/pkg/zlog"
	"github.com/ochinchina/go-daemon"
)

// Daemonize run this process in daemon mode
func Daemonize(logfile string, proc func()) {
	context := daemon.Context{LogFileName: logfile, PidFileName: "supervisord.pid"}

	child, err := context.Reborn()
	if err != nil {
		context := daemon.Context{}
		child, err = context.Reborn()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Fatal("Unable to run")
		}
	}
	if child != nil {
		return
	}
	defer context.Release()
	proc()
}
