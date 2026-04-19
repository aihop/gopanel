package helper

import "time"

type Config struct {
	SocketPath         string
	BaseDir            string
	GoPanelServiceName string
	GoPanelBinaryPath  string
	GoPanelConfigPath  string
	GoPanelPidfilePath string
	ActionTimeout      time.Duration
	LockTimeout        time.Duration
	FileRoots          []string
	AllowRootFS        bool
	MaxFileReadBytes   int64
	MaxFileWriteBytes  int64
}
