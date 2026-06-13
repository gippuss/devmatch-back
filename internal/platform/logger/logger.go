package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	parsedLevel := zapcore.InfoLevel
	if err := parsedLevel.UnmarshalText([]byte(level)); err == nil {
		cfg.Level = zap.NewAtomicLevelAt(parsedLevel)
	}

	return cfg.Build()
}
