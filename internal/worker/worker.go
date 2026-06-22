package worker

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/lupppig/forge-vod/internal/repository/redis"
	"github.com/lupppig/forge-vod/internal/repository/video"
)

// Worker consumes transcode jobs and runs the processing pipeline for each,
// updating the video's status as it goes.
type Worker struct {
	consumer *redis.Consumer
	pipeline *Pipeline
	videos   *video.Repository
	log      *slog.Logger
	scratch  string // base scratch directory for per-job work dirs
}

// New builds a Worker.
func New(consumer *redis.Consumer, pipeline *Pipeline, videos *video.Repository, log *slog.Logger, scratch string) *Worker {
	return &Worker{consumer: consumer, pipeline: pipeline, videos: videos, log: log, scratch: scratch}
}

// Run reads jobs until ctx is cancelled, processing each one.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.consumer.EnsureGroup(ctx); err != nil {
		return err
	}
	w.log.Info("worker started, waiting for jobs")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		job, err := w.consumer.Read(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			w.log.Error("read job failed", slog.Any("error", err))
			continue
		}
		if job == nil {
			continue // block elapsed with no new jobs
		}
		w.handle(ctx, job)
	}
}

// handle processes a single job: marks processing, runs the pipeline, marks
// ready or failed, and acks the stream entry.
func (w *Worker) handle(ctx context.Context, job *redis.Job) {
	log := w.log.With(slog.String("video_id", job.VideoID), slog.String("job_id", job.ID))
	log.Info("job received", slog.String("object_key", job.ObjectKey))

	vid, err := uuid.Parse(job.VideoID)
	if err != nil {
		log.Error("invalid video id, acking to discard", slog.Any("error", err))
		_ = w.consumer.Ack(ctx, job.ID)
		return
	}

	if _, err := w.videos.UpdateStatus(ctx, vid, video.StatusProcessing); err != nil {
		log.Error("failed to mark processing", slog.Any("error", err))
		return // do not ack; let it be retried
	}

	workDir := filepath.Join(w.scratch, job.VideoID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		log.Error("failed to create work dir", slog.Any("error", err))
		return
	}
	defer os.RemoveAll(workDir)

	_, err = w.pipeline.Process(ctx, job.VideoID, job.ObjectKey, workDir, job.VideoID)
	if err != nil {
		log.Error("pipeline failed", slog.Any("error", err))
		if _, uerr := w.videos.UpdateStatus(ctx, vid, video.StatusFailed); uerr != nil {
			log.Error("failed to mark failed", slog.Any("error", uerr))
		}
		_ = w.consumer.Ack(ctx, job.ID)
		return
	}

	if _, err := w.videos.UpdateStatus(ctx, vid, video.StatusReady); err != nil {
		log.Error("failed to mark ready", slog.Any("error", err))
		return
	}
	if err := w.consumer.Ack(ctx, job.ID); err != nil {
		log.Error("failed to ack job", slog.Any("error", err))
		return
	}
	log.Info("job completed")
}
