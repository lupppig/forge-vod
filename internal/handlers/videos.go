package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lupppig/forge-vod/internal/api"
	"github.com/lupppig/forge-vod/internal/middleware"
	"github.com/lupppig/forge-vod/internal/repository/redis"
	"github.com/lupppig/forge-vod/internal/repository/storage"
	"github.com/lupppig/forge-vod/internal/repository/video"
)

const presignExpiry = 15 * time.Minute

type VideoHandler struct {
	repo  *video.Repository
	store *storage.Store
	queue *redis.Queue
}

func NewVideoHandler(repo *video.Repository, store *storage.Store, queue *redis.Queue) *VideoHandler {
	return &VideoHandler{repo: repo, store: store, queue: queue}
}

func (h *VideoHandler) CreateVideo(ctx context.Context, request api.CreateVideoRequestObject) (api.CreateVideoResponseObject, error) {
	log := middleware.Logger(ctx).With(slog.String("op", "create_video"))

	if request.Body == nil || strings.TrimSpace(request.Body.Title) == "" || strings.TrimSpace(request.Body.Filename) == "" {
		log.Warn("rejected: missing title or filename")
		return api.CreateVideo400JSONResponse{ErrorJSONResponse: api.ErrorJSONResponse{
			Message: "title and filename are required",
		}}, nil
	}

	id := uuid.New()
	objectKey := fmt.Sprintf("%s%s", id.String(), path.Ext(request.Body.Filename))
	log = log.With(slog.String("video_id", id.String()), slog.String("object_key", objectKey))
	log.Info("initializing upload", slog.String("title", request.Body.Title))

	v := &video.Video{
		ID:        id,
		Title:     request.Body.Title,
		ObjectKey: objectKey,
		Status:    video.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.repo.Create(ctx, v); err != nil {
		log.Error("failed to persist video record", slog.Any("error", err))
		return nil, err
	}
	log.Info("video record created", slog.String("status", string(v.Status)))

	uploadURL, err := h.store.PresignedPutRaw(ctx, objectKey, presignExpiry)
	if err != nil {
		log.Error("failed to generate presigned url", slog.Any("error", err))
		return nil, err
	}
	log.Info("presigned upload url generated", slog.Duration("expires_in", presignExpiry))

	return api.CreateVideo201JSONResponse{
		Video:     toAPIVideo(v),
		UploadUrl: uploadURL,
		ExpiresIn: int(presignExpiry.Seconds()),
	}, nil
}

func (h *VideoHandler) GetVideo(ctx context.Context, request api.GetVideoRequestObject) (api.GetVideoResponseObject, error) {
	log := middleware.Logger(ctx).With(slog.String("op", "get_video"), slog.String("video_id", request.Id.String()))

	v, err := h.repo.GetByID(ctx, request.Id)
	if errors.Is(err, sql.ErrNoRows) {
		log.Warn("video not found")
		return api.GetVideo404JSONResponse{ErrorJSONResponse: api.ErrorJSONResponse{Message: "video not found"}}, nil
	}
	if err != nil {
		log.Error("failed to fetch video", slog.Any("error", err))
		return nil, err
	}
	log.Info("video fetched", slog.String("status", string(v.Status)))
	return api.GetVideo200JSONResponse(toAPIVideo(v)), nil
}

func (h *VideoHandler) ListVideos(ctx context.Context, request api.ListVideosRequestObject) (api.ListVideosResponseObject, error) {
	log := middleware.Logger(ctx).With(slog.String("op", "list_videos"))

	videos, err := h.repo.List(ctx)
	if err != nil {
		log.Error("failed to list videos", slog.Any("error", err))
		return nil, err
	}
	out := make([]api.Video, 0, len(videos))
	for i := range videos {
		out = append(out, toAPIVideo(&videos[i]))
	}
	log.Info("videos listed", slog.Int("count", len(out)))
	return api.ListVideos200JSONResponse(out), nil
}

func (h *VideoHandler) CompleteVideo(ctx context.Context, request api.CompleteVideoRequestObject) (api.CompleteVideoResponseObject, error) {
	log := middleware.Logger(ctx).With(slog.String("op", "complete_video"), slog.String("video_id", request.Id.String()))

	v, err := h.repo.GetByID(ctx, request.Id)
	if errors.Is(err, sql.ErrNoRows) {
		log.Warn("video not found")
		return api.CompleteVideo404JSONResponse{ErrorJSONResponse: api.ErrorJSONResponse{Message: "video not found"}}, nil
	}
	if err != nil {
		log.Error("failed to fetch video", slog.Any("error", err))
		return nil, err
	}

	updated, err := h.repo.UpdateStatus(ctx, v.ID, video.StatusUploaded)
	if err != nil {
		log.Error("failed to update status", slog.Any("error", err))
		return nil, err
	}
	log.Info("upload marked complete", slog.String("status", string(updated.Status)))

	if err := h.queue.EnqueueTranscode(ctx, updated.ID.String(), updated.ObjectKey); err != nil {
		log.Error("failed to enqueue transcode job", slog.Any("error", err))
		return nil, err
	}
	log.Info("transcode job enqueued",
		slog.String("stream", redis.TranscodeStream),
		slog.String("object_key", updated.ObjectKey),
	)

	return api.CompleteVideo200JSONResponse(toAPIVideo(updated)), nil
}

func toAPIVideo(v *video.Video) api.Video {
	return api.Video{
		Id:        v.ID,
		Title:     v.Title,
		Status:    api.VideoStatus(v.Status),
		ObjectKey: v.ObjectKey,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}
