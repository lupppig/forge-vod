package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string

	RedisURL string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	MinioRawBucket string
	MinioHLSBucket string

	// PublicStreamURL is the base URL under which the HLS bucket is publicly
	// served (e.g. a CDN or the MinIO endpoint). Absolute playback URLs are
	// built as <PublicStreamURL>/<object key>.
	PublicStreamURL string

	// KeyURLBase is the base URL of the API that serves AES-128 keys. The
	// per-video key URI embedded in encrypted playlists is
	// <KeyURLBase>/videos/<id>/key.
	KeyURLBase string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		MinioEndpoint:  os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey: os.Getenv("MINIO_ROOT_USER"),
		MinioSecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioUseSSL:    getBool("MINIO_USE_SSL", false),
		MinioRawBucket: getEnv("MINIO_RAW_BUCKET", "videos"),
		MinioHLSBucket: getEnv("MINIO_HLS_BUCKET", "streams"),
	}

	cfg.PublicStreamURL = strings.TrimRight(getEnv("PUBLIC_STREAM_URL", defaultPublicStreamURL(cfg)), "/")
	cfg.KeyURLBase = strings.TrimRight(getEnv("KEY_URL_BASE", "http://localhost:8080"), "/")

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if c.MinioEndpoint == "" {
		missing = append(missing, "MINIO_ENDPOINT")
	}
	if c.MinioAccessKey == "" {
		missing = append(missing, "MINIO_ROOT_USER")
	}
	if c.MinioSecretKey == "" {
		missing = append(missing, "MINIO_ROOT_PASSWORD")
	}
	if len(missing) > 0 {
		return fmt.Errorf("config: missing required environment variables: %v", missing)
	}
	return nil
}

// defaultPublicStreamURL derives a sensible public base URL for the HLS bucket
// from the MinIO settings when PUBLIC_STREAM_URL is not set: serving the bucket
// directly off the MinIO endpoint, e.g. http://localhost:9000/streams.
func defaultPublicStreamURL(c *Config) string {
	scheme := "http"
	if c.MinioUseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, c.MinioEndpoint, c.MinioHLSBucket)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
