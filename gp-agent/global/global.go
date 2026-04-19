package global

import (
	"time"

	"go.uber.org/zap"
)

type Config struct {
	BaseDir      string
	RunDir       string
	LogDir       string
	ConfDir      string
	SocketPath   string
	EnableCaddy  bool
	EnableDaemon bool
	StartedAt    time.Time
}

var (
	CONF Config
	LOG  *zap.Logger
)
