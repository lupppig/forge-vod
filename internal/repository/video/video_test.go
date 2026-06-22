package video

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lupppig/forge-vod/internal/repository/postgres"
)

// testRepo connects to a Postgres test database, skipping if none is reachable.
// It honours DATABASE_URL, falling back to the local compose default.
func testRepo(t *testing.T) *Repository {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://forge:forge_secret@localhost:5433/forge_vod?sslmode=disable"
	}
	ctx := context.Background()
	db, err := postgres.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres not available (%v); skipping repository test", err)
	}
	t.Cleanup(func() { db.Close() })

	r := NewRepository(db)
	if err := r.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return r
}

func TestSetPlaybackPersistsMetadata(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()

	v := &Video{
		ID:        uuid.New(),
		Title:     "playback test",
		ObjectKey: "raw.mp4",
		Status:    StatusUploaded,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.Create(ctx, v); err != nil {
		t.Fatalf("create: %v", err)
	}

	p := Playback{
		MasterKey:     v.ID.String() + "/master.m3u8",
		ThumbnailKey:  v.ID.String() + "/thumbnail.jpg",
		StoryboardKey: v.ID.String() + "/storyboard.vtt",
		EncKeyHex:     "00112233445566778899aabbccddeeff",
		Duration:      42.5,
	}
	updated, err := r.SetPlayback(ctx, v.ID, StatusReady, p)
	if err != nil {
		t.Fatalf("SetPlayback: %v", err)
	}
	if updated.Status != StatusReady {
		t.Errorf("status = %q, want ready", updated.Status)
	}

	got, err := r.GetByID(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.MasterKey != p.MasterKey || got.EncKeyHex != p.EncKeyHex || got.Duration != p.Duration {
		t.Errorf("playback not persisted: %+v", got)
	}
}
