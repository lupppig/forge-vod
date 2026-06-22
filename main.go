package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/lupppig/forge-vod/config"
	"github.com/lupppig/forge-vod/internal/api"
	"github.com/lupppig/forge-vod/internal/handlers"
	"github.com/lupppig/forge-vod/internal/middleware"
	"github.com/lupppig/forge-vod/internal/repository/postgres"
	"github.com/lupppig/forge-vod/internal/repository/redis"
	"github.com/lupppig/forge-vod/internal/repository/storage"
	"github.com/lupppig/forge-vod/internal/repository/video"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("config loaded")

	ctx := context.Background()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	rdb, err := redis.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("redis connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("connected to redis")

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
	logger.Info("connected to minio", slog.String("raw_bucket", cfg.MinioRawBucket), slog.String("hls_bucket", cfg.MinioHLSBucket))

	videoRepo := video.NewRepository(db)
	if err := videoRepo.Migrate(ctx); err != nil {
		logger.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("database migrated")

	queue := redis.NewQueue(rdb)
	videoHandler := handlers.NewVideoHandler(videoRepo, store, queue, cfg.PublicStreamURL)

	r := mux.NewRouter()
	r.Use(middleware.RequestLogger(logger))
	r.HandleFunc("/docs", handlers.DocsHandler).Methods(http.MethodGet)
	r.HandleFunc("/openapi.json", handlers.OpenAPIHandler).Methods(http.MethodGet)
	api.HandlerFromMux(api.NewStrictHandler(videoHandler, nil), r)

	const addr = ":8080"
	srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	logger.Info("server listening", slog.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
