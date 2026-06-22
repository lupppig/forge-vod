package config

import (
	"testing"
)

// setRequired sets the minimum env vars so Load() passes validation.
func setRequired(t *testing.T) {
	t.Helper()
	// Work from an isolated dir so a repo .env is not picked up by godotenv.
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ROOT_USER", "user")
	t.Setenv("MINIO_ROOT_PASSWORD", "secret")
}

func TestLoadDefaultsForPlaybackURLs(t *testing.T) {
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// PublicStreamURL defaults to the MinIO endpoint + HLS bucket.
	if got, want := cfg.PublicStreamURL, "http://localhost:9000/streams"; got != want {
		t.Errorf("PublicStreamURL = %q, want %q", got, want)
	}
	if got, want := cfg.KeyURLBase, "http://localhost:8080"; got != want {
		t.Errorf("KeyURLBase = %q, want %q", got, want)
	}
}

func TestLoadOverridesAndTrimsTrailingSlash(t *testing.T) {
	setRequired(t)
	t.Setenv("PUBLIC_STREAM_URL", "https://cdn.example.com/vod/")
	t.Setenv("KEY_URL_BASE", "https://api.example.com/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.PublicStreamURL, "https://cdn.example.com/vod"; got != want {
		t.Errorf("PublicStreamURL = %q, want %q", got, want)
	}
	if got, want := cfg.KeyURLBase, "https://api.example.com"; got != want {
		t.Errorf("KeyURLBase = %q, want %q", got, want)
	}
}

func TestLoadMissingRequiredFails(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ROOT_USER", "")
	t.Setenv("MINIO_ROOT_PASSWORD", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when required vars are missing")
	}
}
