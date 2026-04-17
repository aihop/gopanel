package zlog

import (
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Fields map[string]interface{}

type Level = zapcore.Level

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	PanicLevel = zapcore.PanicLevel
	FatalLevel = zapcore.FatalLevel
)

type TextFormatter struct {
	DisableColors bool
	FullTimestamp bool
}

type JSONFormatter struct{}

type Logger struct {
	base  *zap.Logger
	sugar *zap.SugaredLogger
}

type Entry struct {
	sugar *zap.SugaredLogger
}

func ParseLevel(level string) (Level, error) {
	var parsed zapcore.Level
	err := parsed.UnmarshalText([]byte(level))
	return parsed, err
}

func New(writer io.Writer, level Level, formatter interface{}) *Logger {
	base := newZapLogger(writer, level, formatter)
	return &Logger{
		base:  base,
		sugar: base.Sugar(),
	}
}

func (l *Logger) Sync() error {
	if l == nil || l.base == nil {
		return nil
	}
	return l.base.Sync()
}

func (l *Logger) Debug(args ...interface{}) { logArgs(l.sugar.Debug, l.sugar.Debugw, args...) }
func (l *Logger) Info(args ...interface{})  { logArgs(l.sugar.Info, l.sugar.Infow, args...) }
func (l *Logger) Warn(args ...interface{})  { logArgs(l.sugar.Warn, l.sugar.Warnw, args...) }
func (l *Logger) Error(args ...interface{}) { logArgs(l.sugar.Error, l.sugar.Errorw, args...) }
func (l *Logger) Panic(args ...interface{}) { logArgs(l.sugar.Panic, l.sugar.Panicw, args...) }
func (l *Logger) Fatal(args ...interface{}) { logArgs(l.sugar.Fatal, l.sugar.Fatalw, args...) }

func (l *Logger) Debugf(template string, args ...interface{}) { l.sugar.Debugf(template, args...) }
func (l *Logger) Infof(template string, args ...interface{})  { l.sugar.Infof(template, args...) }
func (l *Logger) Warnf(template string, args ...interface{})  { l.sugar.Warnf(template, args...) }
func (l *Logger) Errorf(template string, args ...interface{}) { l.sugar.Errorf(template, args...) }
func (l *Logger) Panicf(template string, args ...interface{}) { l.sugar.Panicf(template, args...) }
func (l *Logger) Fatalf(template string, args ...interface{}) { l.sugar.Fatalf(template, args...) }

func (l *Logger) WithFields(fields Fields) *Entry {
	return &Entry{sugar: l.sugar.With(fieldsToArgs(fields)...)}
}

func (e *Entry) Debug(args ...interface{}) { logArgs(e.sugar.Debug, e.sugar.Debugw, args...) }
func (e *Entry) Info(args ...interface{})  { logArgs(e.sugar.Info, e.sugar.Infow, args...) }
func (e *Entry) Warn(args ...interface{})  { logArgs(e.sugar.Warn, e.sugar.Warnw, args...) }
func (e *Entry) Error(args ...interface{}) { logArgs(e.sugar.Error, e.sugar.Errorw, args...) }
func (e *Entry) Panic(args ...interface{}) { logArgs(e.sugar.Panic, e.sugar.Panicw, args...) }
func (e *Entry) Fatal(args ...interface{}) { logArgs(e.sugar.Fatal, e.sugar.Fatalw, args...) }

func (e *Entry) Debugf(template string, args ...interface{}) { e.sugar.Debugf(template, args...) }
func (e *Entry) Infof(template string, args ...interface{})  { e.sugar.Infof(template, args...) }
func (e *Entry) Warnf(template string, args ...interface{})  { e.sugar.Warnf(template, args...) }
func (e *Entry) Errorf(template string, args ...interface{}) { e.sugar.Errorf(template, args...) }
func (e *Entry) Panicf(template string, args ...interface{}) { e.sugar.Panicf(template, args...) }
func (e *Entry) Fatalf(template string, args ...interface{}) { e.sugar.Fatalf(template, args...) }

var stdState = struct {
	mu        sync.RWMutex
	output    io.Writer
	level     Level
	formatter interface{}
	logger    *Logger
}{
	output:    os.Stdout,
	level:     DebugLevel,
	formatter: &TextFormatter{},
}

func init() {
	rebuildStd()
}

func SetOutput(writer io.Writer) {
	stdState.mu.Lock()
	defer stdState.mu.Unlock()
	if writer == nil {
		writer = io.Discard
	}
	stdState.output = writer
	stdState.logger = New(stdState.output, stdState.level, stdState.formatter)
}

func SetLevel(level Level) {
	stdState.mu.Lock()
	defer stdState.mu.Unlock()
	stdState.level = level
	stdState.logger = New(stdState.output, stdState.level, stdState.formatter)
}

func SetFormatter(formatter interface{}) {
	stdState.mu.Lock()
	defer stdState.mu.Unlock()
	stdState.formatter = formatter
	stdState.logger = New(stdState.output, stdState.level, stdState.formatter)
}

func WithFields(fields Fields) *Entry {
	return &Entry{sugar: std().sugar.With(fieldsToArgs(fields)...)}
}

func Debug(args ...interface{}) { std().Debug(args...) }
func Info(args ...interface{})  { std().Info(args...) }
func Warn(args ...interface{})  { std().Warn(args...) }
func Error(args ...interface{}) { std().Error(args...) }
func Panic(args ...interface{}) { std().Panic(args...) }
func Fatal(args ...interface{}) { std().Fatal(args...) }

func Debugf(template string, args ...interface{}) { std().Debugf(template, args...) }
func Infof(template string, args ...interface{})  { std().Infof(template, args...) }
func Warnf(template string, args ...interface{})  { std().Warnf(template, args...) }
func Errorf(template string, args ...interface{}) { std().Errorf(template, args...) }
func Panicf(template string, args ...interface{}) { std().Panicf(template, args...) }
func Fatalf(template string, args ...interface{}) { std().Fatalf(template, args...) }

func std() *Logger {
	stdState.mu.RLock()
	logger := stdState.logger
	stdState.mu.RUnlock()
	return logger
}

func rebuildStd() {
	stdState.logger = New(stdState.output, stdState.level, stdState.formatter)
}

func newZapLogger(writer io.Writer, level Level, formatter interface{}) *zap.Logger {
	if writer == nil {
		writer = io.Discard
	}
	core := zapcore.NewCore(newEncoder(formatter), zapcore.AddSync(writer), level)
	return zap.New(core)
}

func newEncoder(formatter interface{}) zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		},
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	switch formatter.(type) {
	case *JSONFormatter:
		return zapcore.NewJSONEncoder(cfg)
	default:
		cfg.ConsoleSeparator = " "
		return zapcore.NewConsoleEncoder(cfg)
	}
}

func fieldsToArgs(fields Fields) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

func logArgs(plain func(...interface{}), with func(string, ...interface{}), args ...interface{}) {
	if len(args) >= 3 {
		if msg, ok := args[0].(string); ok && len(args[1:])%2 == 0 {
			with(msg, args[1:]...)
			return
		}
	}
	plain(args...)
}
