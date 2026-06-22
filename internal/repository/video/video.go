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
}

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.NewCreateTable().
		Model((*Video)(nil)).
		IfNotExists().
		Exec(ctx)
	return err
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
