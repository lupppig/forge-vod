package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lupppig/forge-vod/config"
	"github.com/lupppig/forge-vod/internal/repository/postgres"
	"github.com/lupppig/forge-vod/internal/repository/redis"
	"github.com/lupppig/forge-vod/internal/repository/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()
	log.Println("connected to postgres")

	rdb, err := redis.New(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()
	log.Println("connected to redis")

	objstore, err := storage.New(ctx, storage.Config{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccessKey,
		SecretKey: cfg.MinioSecretKey,
		UseSSL:    cfg.MinioUseSSL,
		Buckets:   []string{cfg.MinioRawBucket, cfg.MinioHLSBucket},
	})
	if err != nil {
		log.Fatalf("minio: %v", err)
	}
	_ = objstore
	log.Println("connected to minio")

	r := mux.NewRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Println("server started on port: 8080")
	log.Fatal(srv.ListenAndServe())
}
