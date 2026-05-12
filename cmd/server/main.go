package main

import (
	"fmt"
	"os"

	"pipeline-horn/internal/api"
	"pipeline-horn/internal/config"
	applog "pipeline-horn/internal/log"

	"go.uber.org/zap"
)

func main() {
	logger, err := applog.New("server")
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.LoadServerConfig()
	if err != nil {
		logger.Fatal("load server config", zap.Error(err))
	}

	logger.Info(
		"server starting",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := api.NewServer(cfg, logger)
	logger.Info("server listening", zap.String("addr", addr))
	if err := api.NewRouter(server).Run(addr); err != nil {
		logger.Fatal("server stopped", zap.Error(err))
	}
}
