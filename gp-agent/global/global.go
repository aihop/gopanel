package global

import (
	"time"

	"go.uber.org/zap"
)

type Config struct {
	BaseDir    string
	RunDir     string
	LogDir     string
	BackupDir  string
	SocketPath string
	StartedAt  time.Time
}

var (
	CONF Config
	LOG  *zap.Logger
)
