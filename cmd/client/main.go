package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"pipeline-horn/internal/client"
	"pipeline-horn/internal/config"
	applog "pipeline-horn/internal/log"

	"go.uber.org/zap"
)

func main() {
	logger, err := applog.New("client")
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	cfg, err := config.LoadClientConfigFromArgs(os.Args[1:])
	if err != nil {
		logger.Fatal("load client config", zap.Error(err))
	}

	logger.Info(
		"client starting",
		zap.String("server_url", cfg.ServerURL),
		zap.Int("server_port", cfg.ServerPort),
		zap.Bool("accept_invalid_tls", cfg.AcceptInvalidTLS),
		zap.String("sound_path", cfg.SoundPath),
		zap.String("sound_dir", cfg.SoundDir),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := client.Run(ctx, cfg, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("client stopped", zap.Error(err))
	}
}
