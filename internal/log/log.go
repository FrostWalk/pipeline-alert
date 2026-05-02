package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds production JSON logger and binds service identity.
func New(service string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return logger.With(zap.String("service", service)), nil
}
