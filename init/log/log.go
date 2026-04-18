package log

import (
	"io"
	"os"

	"github.com/aihop/gopanel/global"
	rollinglog "github.com/aihop/gopanel/log"
	"github.com/aihop/gopanel/pkg/zlog"

	"github.com/aihop/gopanel/config"
)

const (
	TimeFormatMi       = "2006-01-02 15:04:05.000"
	TimeFormat         = "2006-01-02 15:04:05"
	FileTImeFormat     = "2006-01-02"
	RollingTimePattern = "0 0  * * *"
)

func Init() {
	l := setOutput(global.CONF.LogConfig)
	global.LOG = l
	global.LOG.Info("init logger successfully")
}

func setOutput(config config.LogConfig) *zlog.Logger {
	writer, err := rollinglog.NewWriterFromConfig(&rollinglog.Config{
		LogPath:            global.CONF.System.LogPath,
		FileName:           config.LogName,
		TimeTagFormat:      FileTImeFormat,
		MaxRemain:          config.MaxBackup,
		RollingTimePattern: RollingTimePattern,
		LogSuffix:          config.LogSuffix,
	})
	if err != nil {
		panic(err)
	}
	level, err := zlog.ParseLevel(config.Level)
	if err != nil {
		panic(err)
	}
	fileAndStdoutWriter := io.MultiWriter(writer, os.Stdout)
	return zlog.New(fileAndStdoutWriter, level, &zlog.TextFormatter{})
}
