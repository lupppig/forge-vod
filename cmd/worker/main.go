package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lupppig/forge-vod/config"
	"github.com/lupppig/forge-vod/internal/repository/postgres"
	"github.com/lupppig/forge-vod/internal/repository/redis"
	"github.com/lupppig/forge-vod/internal/repository/storage"
	"github.com/lupppig/forge-vod/internal/repository/video"
	"github.com/lupppig/forge-vod/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := redis.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("redis connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer rdb.Close()

	store, err := storage.New(ctx, storage.Config{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccessKey,
		SecretKey: cfg.MinioSecretKey,
		UseSSL:    cfg.MinioUseSSL,
		RawBucket: cfg.MinioRawBucket,
		HLSBucket: cfg.MinioHLSBucket,
	})
	if err != nil {
		logger.Error("minio connection failed", slog.Any("error", err))
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	consumer := redis.NewConsumer(rdb, "worker-"+hostname)
	videoRepo := video.NewRepository(db)

	pipeline := &worker.Pipeline{
		Store:   store,
		Encoder: worker.NewFFmpegEncoder(),
		Log:     logger,
		KeyURI:  "enc.key",
	}

	w := worker.New(consumer, pipeline, videoRepo, logger, os.TempDir())
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("worker stopped", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("worker shut down cleanly")
}
