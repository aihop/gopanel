package api

import "runtime"

func gpAgentServiceName() string {
	if runtime.GOOS == "darwin" {
		return "io.aihop.gp-agent"
	}
	return "gp-agent.service"
}
