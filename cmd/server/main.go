package main

import (
	"fmt"
	"os"

	"pipeline-horn/internal/api"
	"pipeline-horn/internal/config"
	applog "pipeline-horn/internal/log"
	"pipeline-horn/internal/loghub"
	"pipeline-horn/internal/sounds"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load server config: %v\n", err)
		os.Exit(1)
	}

	serverHub := loghub.NewHub(cfg.LogBroadcastCap)
	piHub := loghub.NewHub(cfg.LogBroadcastCap)

	logger, err := applog.NewWithHub("server", serverHub)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	soundStore, err := sounds.NewStore(cfg.SoundsDir)
	if err != nil {
		logger.Fatal("init sound store", zap.Error(err))
	}

	logger.Info(
		"server starting",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := api.NewServer(cfg, logger, serverHub, piHub, soundStore)
	logger.Info("server listening", zap.String("addr", addr))
	if err := api.NewRouter(server).Run(addr); err != nil {
		logger.Fatal("server stopped", zap.Error(err))
	}
}
