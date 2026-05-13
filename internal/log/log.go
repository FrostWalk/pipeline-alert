package log

import (
	"os"

	"pipeline-horn/internal/loghub"

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

// NewWithHub tees JSON logs to stdout and to hub (for SSE).
func NewWithHub(service string, hub *loghub.Hub) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	enc := zapcore.NewJSONEncoder(cfg.EncoderConfig)
	console := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), cfg.Level)
	broadcast := loghub.NewZapCore(hub, cfg.Level)

	core := zapcore.NewTee(console, broadcast)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger.With(zap.String("service", service)), nil
}
