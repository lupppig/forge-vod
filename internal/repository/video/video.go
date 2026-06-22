package video

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusUploaded   Status = "uploaded"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

type Video struct {
	bun.BaseModel `bun:"table:videos,alias:v"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	Title     string    `bun:"title,notnull"`
	ObjectKey string    `bun:"object_key,notnull"`
	Status    Status    `bun:"status,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`

	// Playback metadata, populated by the worker once processing succeeds. Keys
	// are object keys within the HLS bucket; EncKeyHex is the hex-encoded
	// AES-128 key served by the key endpoint.
	MasterKey     string  `bun:"master_key"`
	ThumbnailKey  string  `bun:"thumbnail_key"`
	StoryboardKey string  `bun:"storyboard_key"`
	EncKeyHex     string  `bun:"enc_key_hex"`
	Duration      float64 `bun:"duration"`
}

// Playback carries the artifacts produced by processing, persisted with
// SetPlayback when a video becomes ready.
type Playback struct {
	MasterKey     string
	ThumbnailKey  string
	StoryboardKey string
	EncKeyHex     string
	Duration      float64
}

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	if _, err := r.db.NewCreateTable().
		Model((*Video)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// CreateTable IfNotExists will not add columns to a pre-existing table, so
	// add the playback columns idempotently for tables created before they
	// existed.
	addColumns := []string{
		"ALTER TABLE videos ADD COLUMN IF NOT EXISTS master_key text NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN IF NOT EXISTS thumbnail_key text NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN IF NOT EXISTS storyboard_key text NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN IF NOT EXISTS enc_key_hex text NOT NULL DEFAULT ''",
		"ALTER TABLE videos ADD COLUMN IF NOT EXISTS duration double precision NOT NULL DEFAULT 0",
	}
	for _, stmt := range addColumns {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Create(ctx context.Context, v *Video) error {
	_, err := r.db.NewInsert().Model(v).Exec(ctx)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Video, error) {
	v := new(Video)
	err := r.db.NewSelect().Model(v).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) List(ctx context.Context) ([]Video, error) {
	var videos []Video
	err := r.db.NewSelect().Model(&videos).Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, err
	}
	return videos, nil
}

// SetPlayback persists playback metadata and sets the status, in one update.
func (r *Repository) SetPlayback(ctx context.Context, id uuid.UUID, status Status, p Playback) (*Video, error) {
	v := &Video{
		ID:            id,
		Status:        status,
		MasterKey:     p.MasterKey,
		ThumbnailKey:  p.ThumbnailKey,
		StoryboardKey: p.StoryboardKey,
		EncKeyHex:     p.EncKeyHex,
		Duration:      p.Duration,
		UpdatedAt:     time.Now(),
	}
	_, err := r.db.NewUpdate().
		Model(v).
		Column("status", "master_key", "thumbnail_key", "storyboard_key", "enc_key_hex", "duration", "updated_at").
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Video, error) {
	v := &Video{ID: id, Status: status, UpdatedAt: time.Now()}
	_, err := r.db.NewUpdate().
		Model(v).
		Column("status", "updated_at").
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return v, nil
}
