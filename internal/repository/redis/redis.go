package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid url: %w", err)
	}

	client := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	return client, nil
}

const TranscodeStream = "transcode:jobs"

type Queue struct {
	client *redis.Client
}

func NewQueue(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// EnqueueTranscode appends a transcode job for the given video to the
// transcode stream. The worker consumes this stream to produce HLS renditions.
func (q *Queue) EnqueueTranscode(ctx context.Context, videoID, objectKey string) error {
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: TranscodeStream,
		Values: map[string]any{
			"video_id":   videoID,
			"object_key": objectKey,
		},
	}).Err()
}
