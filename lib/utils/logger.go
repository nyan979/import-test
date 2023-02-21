package utils

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitZapLogger(loggingLevel string) *zap.Logger {
	// Default to production level
	logLevel := zap.InfoLevel
	isDev := false
	// Set development to TRUE if DEVELOPMENT is set to true, otherwise default to false
	if loggingLevel == "debug" {
		isDev = true
		logLevel = zap.DebugLevel
	}
	encodeConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Development:      isDev,
		Sampling:         nil, // consider exposing this to config
		Encoding:         "console",
		EncoderConfig:    encodeConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := config.Build()
	if err != nil {
		log.Fatal("Unable to create zap logger")
	}
	return logger
}
